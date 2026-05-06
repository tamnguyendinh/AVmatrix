import type {
  Capture,
  CaptureMatch,
  ParsedImport,
  ParsedTypeBinding,
  SupportedLanguages,
} from 'avmatrix-shared';
import type { ScopeCaptureTreeInput } from '../language-provider.js';
import type { SyntaxNode } from '../utils/ast-helpers.js';

type TsJsLanguage = SupportedLanguages.TypeScript | SupportedLanguages.JavaScript;

interface CaptureOptions {
  readonly language: TsJsLanguage;
}

interface ScopeCaptureContext {
  readonly returnTypesByCallableName: ReadonlyMap<string, string>;
  readonly importedLocalNames: ReadonlySet<string>;
}

export function emitTsJsScopeCapturesFromTree(
  input: ScopeCaptureTreeInput,
): readonly CaptureMatch[] {
  const language = input.language as TsJsLanguage;
  const out: CaptureMatch[] = [];
  const context = buildScopeCaptureContext(input.rootNode);

  out.push(scopeMatch('module', input.rootNode));
  walk(input.rootNode, (node) => {
    emitScope(node, out);
    emitDeclaration(node, out, input.filePath);
    emitImport(node, out);
    emitTypeBinding(node, out, context);
    emitReference(node, out);
  });

  return dedupeCaptureMatches(out, { language });
}

export function interpretTsJsImport(match: CaptureMatch): ParsedImport | null {
  const targetRaw = stripQuotes(match['@import.source']?.text ?? '');
  if (targetRaw.length === 0) return null;

  const kind = match['@import.kind']?.text;
  const importedName = match['@import.imported']?.text ?? match['@import.name']?.text ?? '';
  const localName = match['@import.name']?.text ?? importedName;
  const alias = match['@import.alias']?.text;

  if (kind === 'wildcard') return { kind: 'wildcard', targetRaw };

  if (kind === 'namespace') {
    const imported = importedName.length > 0 ? importedName : moduleNameFromTarget(targetRaw);
    return { kind: 'namespace', localName, importedName: imported, targetRaw };
  }

  if (kind === 'reexport') {
    if (localName.length === 0 || importedName.length === 0) return null;
    return {
      kind: 'reexport',
      localName,
      importedName,
      targetRaw,
      ...(alias !== undefined && alias !== importedName ? { alias } : {}),
    };
  }

  if (localName.length === 0 || importedName.length === 0) return null;
  if (alias !== undefined && alias !== importedName) {
    return { kind: 'alias', localName, importedName, alias: localName, targetRaw };
  }
  return { kind: 'named', localName, importedName, targetRaw };
}

export function interpretTsJsTypeBinding(match: CaptureMatch): ParsedTypeBinding | null {
  const boundName = match['@type-binding.name']?.text;
  const rawTypeName = stripTypeAnnotation(match['@type-binding.type']?.text ?? '');
  if (boundName === undefined || boundName.length === 0 || rawTypeName.length === 0) return null;

  const source = match['@type-binding.source']?.text;
  return {
    boundName,
    rawTypeName,
    source:
      source === 'constructor-inferred' ||
      source === 'assignment-inferred' ||
      source === 'return-annotation' ||
      source === 'call-return' ||
      source === 'call-return-element' ||
      source === 'field-access' ||
      source === 'method-return' ||
      source === 'receiver-propagated'
        ? source
        : source === 'parameter-annotation'
          ? 'parameter-annotation'
          : 'annotation',
  };
}

function emitScope(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'class_declaration' || node.type === 'abstract_class_declaration') {
    out.push(scopeMatch('class', node));
    return;
  }
  if (node.type === 'interface_declaration') {
    out.push(scopeMatch('class', node));
    return;
  }
  if (isFunctionScopeNode(node)) {
    out.push(scopeMatch('function', node));
  }
}

function emitDeclaration(node: SyntaxNode, out: CaptureMatch[], filePath: string): void {
  if (node.type === 'class_declaration' || node.type === 'abstract_class_declaration') {
    emitNamedDeclaration(out, node, 'class', node.childForFieldName('name'));
    return;
  }
  if (node.type === 'interface_declaration') {
    emitNamedDeclaration(out, node, 'interface', node.childForFieldName('name'));
    return;
  }
  if (node.type === 'type_alias_declaration') {
    emitNamedDeclaration(out, node, 'typealias', node.childForFieldName('name'));
    return;
  }
  if (node.type === 'enum_declaration') {
    emitNamedDeclaration(out, node, 'enum', node.childForFieldName('name'));
    return;
  }
  if (node.type === 'function_declaration' || node.type === 'function_signature') {
    emitNamedDeclaration(out, node, 'function', node.childForFieldName('name'), undefined, {
      returnType: returnTypeNameForCallable(node),
    });
    return;
  }
  if (
    node.type === 'method_definition' ||
    node.type === 'abstract_method_signature' ||
    node.type === 'method_signature'
  ) {
    const nameNode = node.childForFieldName('name');
    const ownerName = ownerDeclarationNameFor(node);
    const qualifiedName =
      ownerName !== undefined && nameNode !== null ? `${ownerName}.${nameNode.text}` : undefined;
    emitNamedDeclaration(
      out,
      node,
      nameNode?.text === 'constructor' ? 'constructor' : 'method',
      nameNode,
      ownerDefIdFor(node, filePath),
      { returnType: returnTypeNameForCallable(node), qualifiedName },
    );
    if (ownerName !== undefined)
      out.push(syntheticTypeBindingMatch(node, 'this', ownerName, 'annotation'));
    return;
  }
  if (node.type === 'public_field_definition') {
    const nameNode = node.childForFieldName('name');
    const ownerName = ownerDeclarationNameFor(node);
    const qualifiedName =
      ownerName !== undefined && nameNode !== null ? `${ownerName}.${nameNode.text}` : undefined;
    emitNamedDeclaration(out, node, 'property', nameNode, ownerDefIdFor(node, filePath), {
      declaredType: declaredTypeNameForNode(node),
      qualifiedName,
    });
    return;
  }
  if (node.type === 'property_signature') {
    const nameNode = node.childForFieldName('name');
    const ownerName = ownerDeclarationNameFor(node);
    if (ownerName === undefined) return;
    const qualifiedName = nameNode !== null ? `${ownerName}.${nameNode.text}` : undefined;
    emitNamedDeclaration(out, node, 'property', nameNode, ownerDefIdFor(node, filePath), {
      declaredType: declaredTypeNameForNode(node),
      qualifiedName,
    });
    return;
  }
  if (node.type === 'variable_declarator') {
    const nameNode = node.childForFieldName('name');
    if (nameNode?.type !== 'identifier') return;
    const value = node.childForFieldName('value');
    emitNamedDeclaration(
      out,
      node,
      isFunctionExpression(value) ? 'function' : 'variable',
      nameNode,
      undefined,
      {
        returnType: value === null ? undefined : returnTypeNameForCallable(value),
        declaredType: declaredTypeNameForNode(node),
      },
    );
  }
}

function emitImport(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'import_statement') {
    const sourceNode = node.childForFieldName('source');
    if (sourceNode === null) return;
    const importClause = firstNamedChildOfType(node, 'import_clause');
    if (importClause === undefined) return;

    const namespaceImport = firstDescendantOfType(importClause, 'namespace_import');
    if (namespaceImport !== undefined) {
      const localName = firstDescendantOfType(namespaceImport, 'identifier')?.text;
      if (localName !== undefined) {
        out.push(
          importMatch(node, sourceNode, {
            kind: 'namespace',
            name: localName,
            imported: moduleNameFromTarget(stripQuotes(sourceNode.text)),
          }),
        );
      }
      return;
    }

    const defaultName = directNamedChildOfType(importClause, 'identifier')?.text;
    if (defaultName !== undefined) {
      out.push(
        importMatch(node, sourceNode, {
          kind: 'named',
          name: defaultName,
          imported: 'default',
        }),
      );
    }

    for (const specifier of descendantsOfType(importClause, 'import_specifier')) {
      const names = namedIdentifierChildren(specifier);
      const imported = specifier.childForFieldName('name')?.text ?? names[0]?.text;
      if (imported === undefined) continue;
      const alias = names.length > 1 ? names[names.length - 1]!.text : undefined;
      out.push(
        importMatch(node, sourceNode, {
          kind: alias !== undefined && alias !== imported ? 'alias' : 'named',
          name: alias ?? imported,
          imported,
          alias,
        }),
      );
    }
    return;
  }

  if (node.type !== 'export_statement') return;
  const sourceNode = node.childForFieldName('source');
  if (sourceNode === null) return;

  if (node.text.includes('*')) {
    out.push(importMatch(node, sourceNode, { kind: 'wildcard' }));
    return;
  }

  for (const specifier of descendantsOfType(node, 'export_specifier')) {
    const names = namedIdentifierChildren(specifier);
    const imported = specifier.childForFieldName('name')?.text ?? names[0]?.text;
    if (imported === undefined) continue;
    const alias = names.length > 1 ? names[names.length - 1]!.text : undefined;
    out.push(
      importMatch(node, sourceNode, {
        kind: 'reexport',
        name: alias ?? imported,
        imported,
        alias,
      }),
    );
  }
}

function importedLocalNamesFromStatement(node: SyntaxNode): readonly string[] {
  const importClause = firstNamedChildOfType(node, 'import_clause');
  if (importClause === undefined) return [];

  const out: string[] = [];
  const defaultName = directNamedChildOfType(importClause, 'identifier')?.text;
  if (defaultName !== undefined) out.push(defaultName);

  for (const specifier of descendantsOfType(importClause, 'import_specifier')) {
    const names = namedIdentifierChildren(specifier);
    const imported = specifier.childForFieldName('name')?.text ?? names[0]?.text;
    if (imported === undefined) continue;
    const alias = names.length > 1 ? names[names.length - 1]!.text : undefined;
    out.push(alias ?? imported);
  }

  return out;
}

function emitTypeBinding(
  node: SyntaxNode,
  out: CaptureMatch[],
  context: ScopeCaptureContext,
): void {
  if (node.type === 'required_parameter' || node.type === 'optional_parameter') {
    const nameNode = node.childForFieldName('pattern') ?? firstIdentifierChild(node);
    const typeNode = node.childForFieldName('type');
    if (nameNode !== null && typeNode !== null) {
      out.push(typeBindingMatch(node, nameNode, typeNode, 'parameter-annotation'));
      emitTypeReferenceMatches(typeNode, out);
    }
    return;
  }

  if (node.type === 'public_field_definition') {
    const nameNode = node.childForFieldName('name');
    const typeNode = node.childForFieldName('type');
    if (nameNode !== null && typeNode !== null) {
      out.push(typeBindingMatch(node, nameNode, typeNode, 'annotation'));
      emitTypeReferenceMatches(typeNode, out);
    }
    return;
  }

  if (node.type === 'property_signature') {
    const nameNode = node.childForFieldName('name');
    const typeNode = node.childForFieldName('type');
    if (nameNode !== null && typeNode !== null) {
      out.push(typeBindingMatch(node, nameNode, typeNode, 'annotation'));
      emitTypeReferenceMatches(typeNode, out);
    }
    return;
  }

  if (node.type === 'variable_declarator') {
    const nameNode = node.childForFieldName('name');
    if (nameNode?.type === 'object_pattern') {
      emitObjectPatternFieldBindings(node, nameNode, out, context);
      return;
    }
    if (nameNode?.type !== 'identifier') return;

    const typeNode = node.childForFieldName('type');
    if (typeNode !== null) {
      out.push(typeBindingMatch(node, nameNode, typeNode, 'annotation'));
      emitTypeReferenceMatches(typeNode, out);
      return;
    }

    const ctorName = constructorNameFromValue(node.childForFieldName('value'));
    if (ctorName !== undefined) {
      out.push(inferredTypeBindingMatch(node, nameNode, ctorName, 'constructor-inferred'));
      return;
    }

    const returnTypeName = returnTypeNameFromCallValue(
      node.childForFieldName('value'),
      context.returnTypesByCallableName,
    );
    if (returnTypeName !== undefined) {
      out.push(inferredTypeBindingMatch(node, nameNode, returnTypeName, 'return-annotation'));
      return;
    }

    const callName = callNameFromCallValue(node.childForFieldName('value'));
    if (callName !== undefined && context.importedLocalNames.has(callName)) {
      out.push(inferredTypeBindingMatch(node, nameNode, callName, 'call-return'));
      return;
    }

    const methodReturn = memberMethodNameFromCallValue(node.childForFieldName('value'));
    if (methodReturn !== undefined) {
      out.push(inferredTypeBindingMatch(node, nameNode, methodReturn, 'method-return'));
      return;
    }

    const fieldAccess = memberFieldNameFromValue(node.childForFieldName('value'));
    if (fieldAccess !== undefined) {
      out.push(inferredTypeBindingMatch(node, nameNode, fieldAccess, 'field-access'));
      return;
    }

    const propagatedReceiver = receiverNameFromCopyValue(node.childForFieldName('value'));
    if (propagatedReceiver !== undefined && propagatedReceiver !== nameNode.text) {
      out.push(inferredTypeBindingMatch(node, nameNode, propagatedReceiver, 'receiver-propagated'));
    }
  }

  if (node.type === 'for_in_statement' && isForOfStatement(node)) {
    const nameNode = forOfBindingNameNode(node.childForFieldName('left'));
    if (nameNode === undefined) return;
    const rightNode = node.childForFieldName('right');
    const callName = callNameFromCallValue(rightNode);
    if (
      callName !== undefined &&
      (context.returnTypesByCallableName.has(callName) || context.importedLocalNames.has(callName))
    ) {
      out.push(inferredTypeBindingMatch(node, nameNode, callName, 'call-return-element'));
      return;
    }
    const receiverName = receiverNameFromCopyValue(rightNode);
    if (receiverName !== undefined) {
      out.push(inferredTypeBindingMatch(node, nameNode, receiverName, 'call-return-element'));
    }
    return;
  }

  if (node.type === 'type_alias_declaration') {
    const valueNode = node.childForFieldName('value');
    if (valueNode !== null) emitTypeReferenceMatches(valueNode, out);
    return;
  }

  if (isFunctionScopeNode(node)) {
    const returnTypeNode = node.childForFieldName('return_type');
    if (returnTypeNode !== null) emitTypeReferenceMatches(returnTypeNode, out);
  }
}

function emitObjectPatternFieldBindings(
  node: SyntaxNode,
  nameNode: SyntaxNode,
  out: CaptureMatch[],
  context: ScopeCaptureContext,
): void {
  const receiver = objectPatternReceiverName(node, out, context);
  if (receiver === undefined) return;

  for (let index = 0; index < nameNode.namedChildCount; index++) {
    const child = nameNode.namedChild(index);
    if (child === null) continue;

    if (child.type === 'shorthand_property_identifier_pattern') {
      out.push(inferredTypeBindingMatch(child, child, `${receiver}.${child.text}`, 'field-access'));
      continue;
    }

    if (child.type !== 'pair_pattern') continue;
    const keyNode = child.childForFieldName('key');
    const valueNode = child.childForFieldName('value');
    if (keyNode === null || valueNode?.type !== 'identifier') continue;
    out.push(
      inferredTypeBindingMatch(child, valueNode, `${receiver}.${keyNode.text}`, 'field-access'),
    );
  }
}

function objectPatternReceiverName(
  node: SyntaxNode,
  out: CaptureMatch[],
  context: ScopeCaptureContext,
): string | undefined {
  const value = node.childForFieldName('value');
  const receiver = receiverNameFromCopyValue(value);
  if (receiver !== undefined) return receiver;
  if (value === null) return undefined;

  const callExpression = callExpressionFromValue(value);
  if (callExpression === undefined) return undefined;
  const syntheticReceiver = syntheticDestructureReceiverName(callExpression);

  const returnTypeName = returnTypeNameFromCallValue(value, context.returnTypesByCallableName);
  if (returnTypeName !== undefined) {
    out.push(
      syntheticTypeBindingMatch(
        callExpression,
        syntheticReceiver,
        returnTypeName,
        'return-annotation',
      ),
    );
    return syntheticReceiver;
  }

  const callName = callNameFromCallValue(value);
  if (callName !== undefined && context.importedLocalNames.has(callName)) {
    out.push(syntheticTypeBindingMatch(callExpression, syntheticReceiver, callName, 'call-return'));
    return syntheticReceiver;
  }

  const methodReturn = memberMethodNameFromCallValue(value);
  if (methodReturn !== undefined) {
    out.push(
      syntheticTypeBindingMatch(callExpression, syntheticReceiver, methodReturn, 'method-return'),
    );
    return syntheticReceiver;
  }

  return undefined;
}

function buildScopeCaptureContext(rootNode: SyntaxNode): ScopeCaptureContext {
  const returnTypesByCallableName = new Map<string, string>();
  const importedLocalNames = new Set<string>();
  walk(rootNode, (node) => {
    if (isFunctionScopeNode(node)) {
      const returnType = returnTypeNameForCallable(node);
      if (returnType === undefined) return;
      const name = callableName(node);
      if (name !== undefined) returnTypesByCallableName.set(name, returnType);
    }

    if (node.type === 'variable_declarator') {
      const name = node.childForFieldName('name');
      const value = node.childForFieldName('value');
      const returnType = value === null ? undefined : returnTypeNameForCallable(value);
      if (name?.type === 'identifier' && isFunctionExpression(value) && returnType !== undefined) {
        returnTypesByCallableName.set(name.text, returnType);
      }
    }

    if (node.type === 'import_statement') {
      for (const localName of importedLocalNamesFromStatement(node)) {
        importedLocalNames.add(localName);
      }
    }
  });
  return { returnTypesByCallableName, importedLocalNames };
}

function returnTypeNameForCallable(node: SyntaxNode): string | undefined {
  const returnTypeNode = node.childForFieldName('return_type');
  if (returnTypeNode === null) return undefined;
  const stripped = stripTypeAnnotation(returnTypeNode.text);
  return stripped.length > 0 ? stripped : undefined;
}

function declaredTypeNameForNode(node: SyntaxNode): string | undefined {
  const typeNode = node.childForFieldName('type');
  if (typeNode === null) return undefined;
  const stripped = stripTypeAnnotation(typeNode.text);
  return stripped.length > 0 ? stripped : undefined;
}

function callableName(node: SyntaxNode): string | undefined {
  const name = node.childForFieldName('name');
  return name?.text;
}

function returnTypeNameFromCallValue(
  value: SyntaxNode | null,
  returnTypesByCallableName: ReadonlyMap<string, string>,
): string | undefined {
  if (value === null) return undefined;
  const expression = callExpressionFromValue(value);
  if (expression === undefined) return undefined;
  const fn = unwrapAwaitExpression(expression.childForFieldName('function') ?? expression);
  if (fn.type === 'identifier') return returnTypesByCallableName.get(fn.text);
  return undefined;
}

function callNameFromCallValue(value: SyntaxNode | null): string | undefined {
  if (value === null) return undefined;
  const expression = callExpressionFromValue(value);
  if (expression === undefined) return undefined;
  const fn = unwrapAwaitExpression(expression.childForFieldName('function') ?? expression);
  return fn.type === 'identifier' ? fn.text : undefined;
}

function callExpressionFromValue(value: SyntaxNode): SyntaxNode | undefined {
  const expression = unwrapAwaitExpression(unwrapExpression(value));
  if (expression.type !== 'call_expression') return undefined;
  return expression;
}

function memberMethodNameFromCallValue(value: SyntaxNode | null): string | undefined {
  if (value === null) return undefined;
  const expression = callExpressionFromValue(value);
  if (expression === undefined) return undefined;
  const fn = unwrapAwaitExpression(expression.childForFieldName('function') ?? expression);
  if (fn.type !== 'member_expression') return undefined;
  const receiver = fn.childForFieldName('object');
  const property = fn.childForFieldName('property');
  if (receiver === null || property === null) return undefined;
  if (receiver.type !== 'identifier' && receiver.type !== 'this') return undefined;
  if (property.type !== 'property_identifier') return undefined;
  return `${receiver.text}.${property.text}`;
}

function memberFieldNameFromValue(value: SyntaxNode | null): string | undefined {
  if (value === null) return undefined;
  const expression = unwrapExpression(value);
  if (expression.type !== 'member_expression') return undefined;
  const receiver = expression.childForFieldName('object');
  const property = expression.childForFieldName('property');
  if (receiver === null || property === null) return undefined;
  if (receiver.type !== 'identifier' && receiver.type !== 'this') return undefined;
  if (property.type !== 'property_identifier') return undefined;
  return `${receiver.text}.${property.text}`;
}

function receiverNameFromCopyValue(value: SyntaxNode | null): string | undefined {
  if (value === null) return undefined;
  const expression = unwrapExpression(value);
  return expression.type === 'identifier' || expression.type === 'this'
    ? expression.text
    : undefined;
}

function syntheticDestructureReceiverName(callExpression: SyntaxNode): string {
  const fn = unwrapAwaitExpression(callExpression.childForFieldName('function') ?? callExpression);
  let base = 'call';
  if (fn.type === 'identifier') {
    base = fn.text;
  } else if (fn.type === 'member_expression') {
    base = fn.childForFieldName('property')?.text ?? base;
  }
  return `__destr_${base.replace(/[^\w]/g, '_')}_${callExpression.startIndex}`;
}

function emitTypeReferenceMatches(typeNode: SyntaxNode, out: CaptureMatch[]): void {
  const names = typeReferenceNameNodes(typeNode);
  for (const name of names) {
    out.push(referenceMatch('type', name, name));
  }
}

function emitReference(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'call_expression') {
    const fn = node.childForFieldName('function');
    if (fn === null) return;
    const args = node.childForFieldName('arguments');
    const arity = countArguments(args);

    const member = unwrapAwaitExpression(fn);
    if (member.type === 'member_expression') {
      const property = member.childForFieldName('property');
      const receiver = member.childForFieldName('object');
      if (property !== null)
        out.push(referenceMatch('call.member', node, property, receiver, arity));
      return;
    }

    if (member.type === 'identifier') {
      out.push(referenceMatch('call.free', node, member, undefined, arity));
    }
    return;
  }

  if (node.type === 'member_expression') {
    if (isCallFunctionMember(node)) return;

    const property = node.childForFieldName('property');
    const receiver = node.childForFieldName('object');
    if (property !== null) {
      out.push(referenceMatch(memberAccessKind(node), node, property, receiver));
    }
    return;
  }

  if (node.type === 'new_expression') {
    const ctor = node.childForFieldName('constructor');
    if (ctor !== null) {
      out.push(
        referenceMatch(
          'call.constructor',
          node,
          ctor,
          undefined,
          countArguments(node.childForFieldName('arguments')),
        ),
      );
    }
    return;
  }

  if (node.type === 'extends_clause') {
    const value = node.childForFieldName('value') ?? firstIdentifierLikeChild(node);
    if (value !== null && value !== undefined) out.push(referenceMatch('inherits', node, value));
    return;
  }

  if (node.type === 'implements_clause') {
    for (const ident of namedIdentifierChildren(node)) {
      out.push(referenceMatch('inherits', ident, ident));
    }
  }
}

function emitNamedDeclaration(
  out: CaptureMatch[],
  node: SyntaxNode,
  kind: string,
  nameNode: SyntaxNode | null,
  ownerId?: string,
  metadata: {
    readonly returnType?: string;
    readonly declaredType?: string;
    readonly qualifiedName?: string;
  } = {},
): void {
  if (nameNode === null) return;
  out.push({
    [`@declaration.${kind}`]: capture(`@declaration.${kind}`, node),
    '@declaration.name': capture('@declaration.name', nameNode),
    ...(ownerId !== undefined
      ? { '@declaration.owner': textCapture('@declaration.owner', node, ownerId) }
      : {}),
    ...(metadata.qualifiedName !== undefined
      ? {
          '@declaration.qualified_name': textCapture(
            '@declaration.qualified_name',
            node,
            metadata.qualifiedName,
          ),
        }
      : {}),
    ...(metadata.returnType !== undefined
      ? {
          '@declaration.return_type': textCapture(
            '@declaration.return_type',
            node,
            metadata.returnType,
          ),
        }
      : {}),
    ...(metadata.declaredType !== undefined
      ? {
          '@declaration.declared_type': textCapture(
            '@declaration.declared_type',
            node,
            metadata.declaredType,
          ),
        }
      : {}),
  });
}

function scopeMatch(kind: 'module' | 'class' | 'function', node: SyntaxNode): CaptureMatch {
  return { [`@scope.${kind}`]: capture(`@scope.${kind}`, node) };
}

function importMatch(
  statement: SyntaxNode,
  sourceNode: SyntaxNode,
  parts: {
    readonly kind: 'named' | 'alias' | 'namespace' | 'reexport' | 'wildcard';
    readonly name?: string;
    readonly imported?: string;
    readonly alias?: string;
  },
): CaptureMatch {
  return {
    '@import.statement': capture('@import.statement', statement),
    '@import.source': capture('@import.source', sourceNode),
    '@import.kind': textCapture('@import.kind', statement, parts.kind),
    ...(parts.name !== undefined
      ? { '@import.name': textCapture('@import.name', statement, parts.name) }
      : {}),
    ...(parts.imported !== undefined
      ? { '@import.imported': textCapture('@import.imported', statement, parts.imported) }
      : {}),
    ...(parts.alias !== undefined
      ? { '@import.alias': textCapture('@import.alias', statement, parts.alias) }
      : {}),
  };
}

function typeBindingMatch(
  anchor: SyntaxNode,
  nameNode: SyntaxNode,
  typeNode: SyntaxNode,
  source: string,
): CaptureMatch {
  return {
    '@type-binding.parameter': capture('@type-binding.parameter', anchor),
    '@type-binding.name': capture('@type-binding.name', nameNode),
    '@type-binding.type': capture('@type-binding.type', typeNode),
    '@type-binding.source': textCapture('@type-binding.source', anchor, source),
  };
}

function inferredTypeBindingMatch(
  anchor: SyntaxNode,
  nameNode: SyntaxNode,
  typeName: string,
  source: string,
): CaptureMatch {
  return {
    '@type-binding.assignment': capture('@type-binding.assignment', anchor),
    '@type-binding.name': capture('@type-binding.name', nameNode),
    '@type-binding.type': textCapture('@type-binding.type', anchor, typeName),
    '@type-binding.source': textCapture('@type-binding.source', anchor, source),
  };
}

function syntheticTypeBindingMatch(
  anchor: SyntaxNode,
  boundName: string,
  typeName: string,
  source: string,
): CaptureMatch {
  return {
    '@type-binding.assignment': capture('@type-binding.assignment', anchor),
    '@type-binding.name': textCapture('@type-binding.name', anchor, boundName),
    '@type-binding.type': textCapture('@type-binding.type', anchor, typeName),
    '@type-binding.source': textCapture('@type-binding.source', anchor, source),
  };
}

function referenceMatch(
  suffix: string,
  anchor: SyntaxNode,
  nameNode: SyntaxNode,
  receiver?: SyntaxNode | null,
  arity?: number,
): CaptureMatch {
  return {
    [`@reference.${suffix}`]: capture(`@reference.${suffix}`, anchor),
    '@reference.name': capture('@reference.name', nameNode),
    ...(receiver !== undefined && receiver !== null
      ? { '@reference.receiver': capture('@reference.receiver', receiver) }
      : {}),
    ...(arity !== undefined
      ? { '@reference.arity': textCapture('@reference.arity', anchor, String(arity)) }
      : {}),
  };
}

function capture(name: string, node: SyntaxNode): Capture {
  return {
    name,
    range: {
      startLine: node.startPosition.row + 1,
      startCol: node.startPosition.column,
      endLine: node.endPosition.row + 1,
      endCol: node.endPosition.column,
    },
    text: node.text,
  };
}

function textCapture(name: string, anchor: SyntaxNode, text: string): Capture {
  return { ...capture(name, anchor), text };
}

function walk(node: SyntaxNode, visit: (node: SyntaxNode) => void): void {
  visit(node);
  for (let i = 0; i < node.namedChildCount; i++) {
    const child = node.namedChild(i);
    if (child !== null) walk(child, visit);
  }
}

function isFunctionScopeNode(node: SyntaxNode): boolean {
  return (
    node.type === 'function_declaration' ||
    node.type === 'function_signature' ||
    node.type === 'method_definition' ||
    node.type === 'abstract_method_signature' ||
    node.type === 'method_signature' ||
    node.type === 'arrow_function' ||
    node.type === 'function_expression'
  );
}

function isFunctionExpression(node: SyntaxNode | null): boolean {
  return node?.type === 'arrow_function' || node?.type === 'function_expression';
}

function isForOfStatement(node: SyntaxNode): boolean {
  for (let i = 0; i < node.childCount; i++) {
    if (node.child(i)?.type === 'of') return true;
  }
  return false;
}

function forOfBindingNameNode(leftNode: SyntaxNode | null): SyntaxNode | undefined {
  if (leftNode?.type === 'identifier') return leftNode;
  if (leftNode?.type !== 'variable_declarator') return undefined;
  const nameNode = leftNode.childForFieldName('name') ?? leftNode.firstNamedChild;
  return nameNode?.type === 'identifier' ? nameNode : undefined;
}

function constructorNameFromValue(node: SyntaxNode | null): string | undefined {
  if (node === null) return undefined;
  const value = unwrapExpression(node);
  if (value.type !== 'new_expression') return undefined;
  const ctor = value.childForFieldName('constructor');
  return ctor?.text;
}

function unwrapExpression(node: SyntaxNode): SyntaxNode {
  let current = node;
  while (
    current.type === 'as_expression' ||
    current.type === 'non_null_expression' ||
    current.type === 'parenthesized_expression'
  ) {
    const next = current.namedChild(0);
    if (next === null) break;
    current = next;
  }
  return current;
}

function unwrapAwaitExpression(node: SyntaxNode): SyntaxNode {
  if (node.type !== 'await_expression') return node;
  return node.namedChild(0) ?? node;
}

function isCallFunctionMember(node: SyntaxNode): boolean {
  const parent = node.parent;
  if (parent?.type !== 'call_expression') return false;
  const fn = parent.childForFieldName('function');
  return fn !== null && isSameSyntaxNode(fn, node);
}

function memberAccessKind(node: SyntaxNode): 'read' | 'write' {
  const parent = node.parent;
  if (parent === null) return 'read';

  if (
    parent.type === 'assignment_expression' ||
    parent.type === 'augmented_assignment_expression'
  ) {
    const left = parent.childForFieldName('left') ?? parent.namedChild(0);
    return left !== null && isSameSyntaxNode(left, node) ? 'write' : 'read';
  }

  if (parent.type === 'update_expression') return 'write';
  return 'read';
}

function isSameSyntaxNode(a: SyntaxNode, b: SyntaxNode): boolean {
  return a.type === b.type && a.startIndex === b.startIndex && a.endIndex === b.endIndex;
}

function countArguments(args: SyntaxNode | null): number | undefined {
  if (args === null) return undefined;
  let count = 0;
  for (let i = 0; i < args.namedChildCount; i++) {
    const child = args.namedChild(i);
    if (child !== null && child.type !== 'comment') count++;
  }
  return count;
}

function firstNamedChildOfType(node: SyntaxNode, type: string): SyntaxNode | undefined {
  for (let i = 0; i < node.namedChildCount; i++) {
    const child = node.namedChild(i);
    if (child?.type === type) return child;
  }
  return undefined;
}

function directNamedChildOfType(node: SyntaxNode, type: string): SyntaxNode | undefined {
  return firstNamedChildOfType(node, type);
}

function firstDescendantOfType(node: SyntaxNode, type: string): SyntaxNode | undefined {
  for (const child of descendantsOfType(node, type)) return child;
  return undefined;
}

function descendantsOfType(node: SyntaxNode, type: string): SyntaxNode[] {
  const out: SyntaxNode[] = [];
  walk(node, (candidate) => {
    if (!isSameSyntaxNode(candidate, node) && candidate.type === type) out.push(candidate);
  });
  return out;
}

function namedIdentifierChildren(node: SyntaxNode): SyntaxNode[] {
  const out: SyntaxNode[] = [];
  for (let i = 0; i < node.namedChildCount; i++) {
    const child = node.namedChild(i);
    if (child !== null && isIdentifierLike(child)) out.push(child);
  }
  return out;
}

function firstIdentifierChild(node: SyntaxNode): SyntaxNode | null {
  return namedIdentifierChildren(node)[0] ?? null;
}

function firstIdentifierLikeChild(node: SyntaxNode): SyntaxNode | undefined {
  for (let i = 0; i < node.namedChildCount; i++) {
    const child = node.namedChild(i);
    if (child !== null && isIdentifierLike(child)) return child;
  }
  return undefined;
}

const BUILTIN_TYPE_NAMES: ReadonlySet<string> = new Set([
  'any',
  'unknown',
  'never',
  'void',
  'string',
  'number',
  'boolean',
  'bigint',
  'symbol',
  'object',
  'undefined',
  'null',
  'true',
  'false',
]);

function typeReferenceNameNodes(typeNode: SyntaxNode): readonly SyntaxNode[] {
  const out: SyntaxNode[] = [];
  walk(typeNode, (candidate) => {
    if (!isIdentifierLike(candidate)) return;
    if (BUILTIN_TYPE_NAMES.has(candidate.text)) return;
    out.push(candidate);
  });
  return out;
}

function isIdentifierLike(node: SyntaxNode): boolean {
  return (
    node.type === 'identifier' ||
    node.type === 'type_identifier' ||
    node.type === 'property_identifier' ||
    node.type === 'private_property_identifier'
  );
}

function ownerDefIdFor(node: SyntaxNode, filePath: string): string | undefined {
  let current = node.parent;
  while (current !== null) {
    const ownerKind = ownerDeclarationKind(current);
    if (ownerKind !== undefined) {
      const nameNode = current.childForFieldName('name');
      if (nameNode === null) return undefined;
      return defId(filePath, current, ownerKind, nameNode.text);
    }
    current = current.parent;
  }
  return undefined;
}

function ownerDeclarationNameFor(node: SyntaxNode): string | undefined {
  let current = node.parent;
  while (current !== null) {
    const ownerKind = ownerDeclarationKind(current);
    if (ownerKind !== undefined) {
      return current.childForFieldName('name')?.text;
    }
    current = current.parent;
  }
  return undefined;
}

function ownerDeclarationKind(node: SyntaxNode): 'Class' | 'Interface' | undefined {
  if (node.type === 'class_declaration' || node.type === 'abstract_class_declaration') {
    return 'Class';
  }
  if (node.type === 'interface_declaration') return 'Interface';
  return undefined;
}

function defId(
  filePath: string,
  node: SyntaxNode,
  type: 'Class' | 'Interface',
  name: string,
): string {
  const range = capture('def', node).range;
  return `def:${filePath}#${range.startLine}:${range.startCol}:${type}:${name}`;
}

function stripQuotes(value: string): string {
  const trimmed = value.trim();
  if (
    (trimmed.startsWith("'") && trimmed.endsWith("'")) ||
    (trimmed.startsWith('"') && trimmed.endsWith('"'))
  ) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}

function stripTypeAnnotation(value: string): string {
  return value.trim().replace(/^:\s*/, '').trim();
}

function moduleNameFromTarget(targetRaw: string): string {
  const normalized = targetRaw.replace(/\\/g, '/');
  const tail = normalized.split('/').filter(Boolean).pop();
  return tail ?? targetRaw;
}

function dedupeCaptureMatches(
  matches: readonly CaptureMatch[],
  _options: CaptureOptions,
): readonly CaptureMatch[] {
  const seen = new Set<string>();
  const out: CaptureMatch[] = [];
  for (const match of matches) {
    const key = Object.entries(match)
      .map(([name, cap]) => `${name}:${cap.range.startLine}:${cap.range.startCol}:${cap.text}`)
      .sort()
      .join('|');
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(match);
  }
  return out;
}
