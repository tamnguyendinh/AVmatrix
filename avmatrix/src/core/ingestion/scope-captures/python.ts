import type { Capture, CaptureMatch, ParsedImport, ParsedTypeBinding } from 'avmatrix-shared';
import type { ScopeCaptureTreeInput } from '../language-provider.js';
import type { SyntaxNode } from '../utils/ast-helpers.js';

interface ScopeCaptureContext {
  readonly propertyTypes: ReadonlyMap<string, string>;
  readonly emittedProperties: Set<string>;
}

const PYTHON_BUILTIN_TYPE_NAMES: ReadonlySet<string> = new Set([
  'Any',
  'None',
  'bool',
  'bytes',
  'dict',
  'float',
  'int',
  'list',
  'object',
  'set',
  'str',
  'tuple',
]);

export function emitPythonScopeCapturesFromTree(
  input: ScopeCaptureTreeInput,
): readonly CaptureMatch[] {
  const out: CaptureMatch[] = [scopeMatch('module', input.rootNode)];
  const context: ScopeCaptureContext = {
    propertyTypes: buildPropertyTypes(input.rootNode),
    emittedProperties: new Set(),
  };

  walk(input.rootNode, (node) => {
    emitScope(node, out);
    emitDeclaration(node, out, input.filePath, context);
    emitImport(node, out);
    emitTypeBinding(node, out);
    emitReference(node, out);
  });

  return dedupeCaptureMatches(out);
}

export function interpretPythonImport(match: CaptureMatch): ParsedImport | null {
  const targetRaw = match['@import.source']?.text?.trim();
  const kind = match['@import.kind']?.text;
  if (targetRaw === undefined || targetRaw.length === 0) return null;

  if (kind === 'wildcard') return { kind: 'wildcard', targetRaw };

  const importedName = match['@import.imported']?.text;
  const localName = match['@import.name']?.text ?? importedName;

  if (kind === 'namespace') {
    const imported = importedName ?? moduleTail(targetRaw);
    return {
      kind: 'namespace',
      localName: localName ?? imported,
      importedName: imported,
      targetRaw,
    };
  }

  if (localName === undefined || importedName === undefined) return null;
  if (kind === 'alias')
    return { kind: 'alias', localName, importedName, alias: localName, targetRaw };
  return { kind: 'named', localName, importedName, targetRaw };
}

export function interpretPythonTypeBinding(match: CaptureMatch): ParsedTypeBinding | null {
  const boundName = match['@type-binding.name']?.text;
  const rawTypeName = normalizePythonTypeName(match['@type-binding.type']?.text ?? '');
  if (boundName === undefined || boundName.length === 0 || rawTypeName === undefined) return null;

  const source = match['@type-binding.source']?.text;
  return {
    boundName,
    rawTypeName,
    source:
      source === 'self' ||
      source === 'parameter-annotation' ||
      source === 'constructor-inferred' ||
      source === 'assignment-inferred' ||
      source === 'return-annotation' ||
      source === 'call-return' ||
      source === 'call-return-element' ||
      source === 'field-access' ||
      source === 'method-return' ||
      source === 'receiver-propagated'
        ? source
        : 'annotation',
  };
}

function emitScope(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'class_definition') {
    out.push(scopeMatch('class', node));
    return;
  }
  if (node.type === 'function_definition') out.push(scopeMatch('function', node));
}

function emitDeclaration(
  node: SyntaxNode,
  out: CaptureMatch[],
  filePath: string,
  context: ScopeCaptureContext,
): void {
  if (node.type === 'class_definition') {
    emitNamedDeclaration(out, node, 'class', node.childForFieldName('name'));
    return;
  }

  if (node.type === 'function_definition') {
    const nameNode = node.childForFieldName('name');
    const ownerName = ownerClassNameFor(node);
    const ownerId = ownerDefIdFor(node, filePath);
    const qualifiedName =
      ownerName !== undefined && nameNode !== null ? `${ownerName}.${nameNode.text}` : undefined;
    emitNamedDeclaration(
      out,
      node,
      ownerName === undefined ? 'function' : 'method',
      nameNode,
      ownerId,
      {
        returnType: returnTypeNameForCallable(node),
        qualifiedName,
      },
    );
    return;
  }

  if (node.type !== 'assignment') return;
  const field = selfAttributeAssignment(node);
  if (field === undefined) return;
  const ownerName = ownerClassNameFor(node);
  if (ownerName === undefined) return;
  const propertyKey = `${ownerName}.${field.nameNode.text}`;
  if (context.emittedProperties.has(propertyKey)) return;
  context.emittedProperties.add(propertyKey);
  emitNamedDeclaration(out, node, 'property', field.nameNode, ownerDefIdFor(node, filePath), {
    declaredType:
      context.propertyTypes.get(propertyKey) ??
      constructorNameFromValue(node.childForFieldName('right')) ??
      annotatedIdentifierTypeFromAssignment(node),
    qualifiedName: propertyKey,
  });
}

function emitImport(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'import_statement') {
    for (let index = 0; index < node.namedChildCount; index++) {
      const child = node.namedChild(index);
      if (child === null) continue;

      if (child.type === 'aliased_import') {
        const imported = firstDescendantOfType(child, 'dotted_name');
        const alias = child.childForFieldName('alias') ?? lastIdentifierChild(child);
        if (imported === undefined || alias === null || alias === undefined) continue;
        out.push(
          importMatch(node, imported, {
            kind: 'namespace',
            name: alias.text,
            imported: imported.text,
          }),
        );
        continue;
      }

      if (child.type === 'dotted_name' || child.type === 'identifier') {
        out.push(
          importMatch(node, child, {
            kind: 'namespace',
            name: moduleHead(child.text),
            imported: moduleTail(child.text),
          }),
        );
      }
    }
    return;
  }

  if (node.type !== 'import_from_statement') return;
  const sourceNode = node.childForFieldName('module_name');
  if (sourceNode === null) return;

  const wildcard = firstNamedChildOfType(node, 'wildcard_import');
  if (wildcard !== undefined) {
    out.push(importMatch(node, sourceNode, { kind: 'wildcard' }));
    return;
  }

  for (let index = 0; index < node.namedChildCount; index++) {
    const child = node.namedChild(index);
    if (child === null || isSameSyntaxNode(child, sourceNode)) continue;

    if (child.type === 'aliased_import') {
      const imported = firstDescendantOfType(child, 'dotted_name');
      const alias = child.childForFieldName('alias') ?? lastIdentifierChild(child);
      if (imported === undefined || alias === null || alias === undefined) continue;
      out.push(
        importMatch(node, sourceNode, {
          kind: 'alias',
          name: alias.text,
          imported: imported.text,
          alias: alias.text,
        }),
      );
      continue;
    }

    if (child.type === 'dotted_name' || child.type === 'identifier') {
      out.push(
        importMatch(node, sourceNode, {
          kind: 'named',
          name: child.text,
          imported: child.text,
        }),
      );
    }
  }
}

function emitTypeBinding(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'function_definition') {
    const ownerName = ownerClassNameFor(node);
    if (ownerName !== undefined) {
      const selfNode = firstParameterIdentifier(node);
      if (selfNode !== undefined && (selfNode.text === 'self' || selfNode.text === 'cls')) {
        out.push(inferredTypeBindingMatch(selfNode, selfNode, ownerName, 'self'));
      }
    }

    const parameters = node.childForFieldName('parameters');
    if (parameters !== null) {
      for (const param of descendantsOfType(parameters, 'typed_parameter')) {
        const nameNode = firstIdentifierChild(param);
        const typeNode = param.childForFieldName('type') ?? lastNamedChild(param);
        if (nameNode !== undefined && typeNode !== null && typeNode !== undefined) {
          out.push(typeBindingMatch(param, nameNode, typeNode, 'parameter-annotation'));
          emitTypeReferenceMatches(typeNode, out);
        }
      }
    }

    const returnType = node.childForFieldName('return_type');
    if (returnType !== null) emitTypeReferenceMatches(returnType, out);
    return;
  }

  if (node.type !== 'assignment') return;
  const left = node.childForFieldName('left');
  const right = node.childForFieldName('right');
  if (left === null) return;

  if (left.type === 'identifier') {
    const ctorName = constructorNameFromValue(right);
    if (ctorName !== undefined) {
      out.push(inferredTypeBindingMatch(node, left, ctorName, 'constructor-inferred'));
      return;
    }

    const receiver = receiverNameFromCopyValue(right);
    if (receiver !== undefined && receiver !== left.text) {
      out.push(inferredTypeBindingMatch(node, left, receiver, 'receiver-propagated'));
      return;
    }

    const fieldAccess = memberFieldNameFromValue(right);
    if (fieldAccess !== undefined) {
      out.push(inferredTypeBindingMatch(node, left, fieldAccess, 'field-access'));
      return;
    }

    const methodReturn = memberMethodNameFromCallValue(right);
    if (methodReturn !== undefined) {
      out.push(inferredTypeBindingMatch(node, left, methodReturn, 'method-return'));
    }
    return;
  }

  const field = selfAttributeAssignment(node);
  if (field === undefined) return;
  const ctorName = constructorNameFromValue(right);
  if (ctorName !== undefined) {
    out.push(inferredTypeBindingMatch(node, field.nameNode, ctorName, 'constructor-inferred'));
    return;
  }
  const annotatedType = annotatedIdentifierTypeFromAssignment(node);
  if (annotatedType !== undefined) {
    out.push(inferredTypeBindingMatch(node, field.nameNode, annotatedType, 'assignment-inferred'));
  }
}

function emitReference(node: SyntaxNode, out: CaptureMatch[]): void {
  if (node.type === 'call') {
    const fn = node.childForFieldName('function');
    if (fn === null) return;
    const args = node.childForFieldName('arguments');
    const arity = countArguments(args);

    if (fn.type === 'attribute') {
      const property = fn.childForFieldName('attribute');
      const receiver = fn.childForFieldName('object');
      if (property !== null)
        out.push(referenceMatch('call.member', node, property, receiver, arity));
      return;
    }

    if (fn.type === 'identifier') {
      out.push(referenceMatch('call.free', node, fn, undefined, arity));
    }
    return;
  }

  if (node.type === 'attribute') {
    if (isCallFunctionAttribute(node)) return;
    const property = node.childForFieldName('attribute');
    const receiver = node.childForFieldName('object');
    if (property !== null)
      out.push(referenceMatch(memberAccessKind(node), node, property, receiver));
    return;
  }

  if (node.type === 'class_definition') {
    const superclasses = node.childForFieldName('superclasses');
    if (superclasses === null) return;
    for (let index = 0; index < superclasses.namedChildCount; index++) {
      const child = superclasses.namedChild(index);
      if (child === null) continue;
      const name = referenceNameNode(child);
      if (name !== undefined) out.push(heritageReferenceMatch('extends', child, name));
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
    readonly kind: 'named' | 'alias' | 'namespace' | 'wildcard';
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

function heritageReferenceMatch(
  heritageKind: 'extends',
  anchor: SyntaxNode,
  nameNode: SyntaxNode,
): CaptureMatch {
  return {
    ...referenceMatch('inherits', anchor, nameNode),
    '@reference.heritage_kind': textCapture('@reference.heritage_kind', anchor, heritageKind),
  };
}

function emitTypeReferenceMatches(typeNode: SyntaxNode, out: CaptureMatch[]): void {
  const name = normalizePythonTypeName(typeNode.text);
  if (name === undefined) return;
  out.push(referenceMatch('type', typeNode, typeNode));
}

function normalizePythonTypeName(raw: string): string | undefined {
  let value = raw.trim();
  if (value.length === 0) return undefined;
  if (value.startsWith(':')) value = value.slice(1).trim();
  value = value.replace(/^['"]|['"]$/g, '');
  const pipeParts = value
    .split('|')
    .map((part) => part.trim())
    .filter((part) => part.length > 0 && part !== 'None');
  if (pipeParts.length === 1) value = pipeParts[0]!;
  const genericStart = value.indexOf('[');
  if (genericStart > 0) value = value.slice(0, genericStart);
  const segments = value.split('.').filter(Boolean);
  value = segments[segments.length - 1] ?? value;
  return /^[A-Za-z_]\w*$/.test(value) && !PYTHON_BUILTIN_TYPE_NAMES.has(value) ? value : undefined;
}

function returnTypeNameForCallable(node: SyntaxNode): string | undefined {
  const returnTypeNode = node.childForFieldName('return_type');
  if (returnTypeNode === null) return undefined;
  return normalizePythonTypeName(returnTypeNode.text);
}

function buildPropertyTypes(rootNode: SyntaxNode): ReadonlyMap<string, string> {
  const out = new Map<string, string>();
  const ambiguous = new Set<string>();

  walk(rootNode, (node) => {
    if (node.type !== 'assignment') return;
    const field = selfAttributeAssignment(node);
    if (field === undefined) return;
    const ownerName = ownerClassNameFor(node);
    if (ownerName === undefined) return;
    const typeName =
      constructorNameFromValue(node.childForFieldName('right')) ??
      annotatedIdentifierTypeFromAssignment(node);
    if (typeName === undefined) return;

    const key = `${ownerName}.${field.nameNode.text}`;
    if (ambiguous.has(key)) return;
    const existing = out.get(key);
    if (existing !== undefined && existing !== typeName) {
      out.delete(key);
      ambiguous.add(key);
      return;
    }
    out.set(key, typeName);
  });

  return out;
}

function annotatedIdentifierTypeFromAssignment(assignment: SyntaxNode): string | undefined {
  const right = assignment.childForFieldName('right');
  if (right?.type !== 'identifier') return undefined;
  const functionNode = ownerFunctionNodeFor(assignment);
  if (functionNode === undefined) return undefined;
  return parameterTypeBindingsForFunction(functionNode).get(right.text);
}

function parameterTypeBindingsForFunction(functionNode: SyntaxNode): ReadonlyMap<string, string> {
  const out = new Map<string, string>();
  const parameters = functionNode.childForFieldName('parameters');
  if (parameters === null) return out;

  for (const param of descendantsOfType(parameters, 'typed_parameter')) {
    const nameNode = firstIdentifierChild(param);
    const typeNode = param.childForFieldName('type') ?? lastNamedChild(param);
    if (nameNode === undefined || typeNode === null || typeNode === undefined) continue;
    const typeName = normalizePythonTypeName(typeNode.text);
    if (typeName !== undefined) out.set(nameNode.text, typeName);
  }

  return out;
}

function constructorNameFromValue(node: SyntaxNode | null): string | undefined {
  const call = callExpressionFromValue(node);
  if (call === undefined) return undefined;
  const fn = call.childForFieldName('function');
  if (fn?.type === 'identifier') return fn.text;
  return undefined;
}

function callExpressionFromValue(node: SyntaxNode | null): SyntaxNode | undefined {
  if (node === null) return undefined;
  const expression = unwrapExpression(node);
  return expression.type === 'call' ? expression : undefined;
}

function memberMethodNameFromCallValue(node: SyntaxNode | null): string | undefined {
  const call = callExpressionFromValue(node);
  if (call === undefined) return undefined;
  const fn = call.childForFieldName('function');
  if (fn?.type !== 'attribute') return undefined;
  const receiver = fn.childForFieldName('object');
  const attribute = fn.childForFieldName('attribute');
  if (receiver === null || attribute === null) return undefined;
  return `${receiver.text}.${attribute.text}`;
}

function memberFieldNameFromValue(node: SyntaxNode | null): string | undefined {
  if (node === null) return undefined;
  const expression = unwrapExpression(node);
  if (expression.type !== 'attribute') return undefined;
  const receiver = expression.childForFieldName('object');
  const attribute = expression.childForFieldName('attribute');
  if (receiver === null || attribute === null) return undefined;
  return `${receiver.text}.${attribute.text}`;
}

function receiverNameFromCopyValue(node: SyntaxNode | null): string | undefined {
  if (node === null) return undefined;
  const expression = unwrapExpression(node);
  return expression.type === 'identifier' ? expression.text : undefined;
}

function unwrapExpression(node: SyntaxNode): SyntaxNode {
  let current = node;
  while (
    current.type === 'parenthesized_expression' ||
    current.type === 'conditional_expression' ||
    current.type === 'as_pattern'
  ) {
    const next = current.namedChild(0);
    if (next === null) break;
    current = next;
  }
  return current;
}

function selfAttributeAssignment(
  assignment: SyntaxNode,
): { readonly nameNode: SyntaxNode } | undefined {
  const left = assignment.childForFieldName('left');
  if (left?.type !== 'attribute') return undefined;
  const receiver = left.childForFieldName('object');
  const nameNode = left.childForFieldName('attribute');
  if (receiver?.type !== 'identifier' || nameNode === null) return undefined;
  if (receiver.text !== 'self' && receiver.text !== 'cls') return undefined;
  return { nameNode };
}

function ownerClassNameFor(node: SyntaxNode): string | undefined {
  const owner = ownerClassNodeFor(node);
  return owner?.childForFieldName('name')?.text;
}

function ownerClassNodeFor(node: SyntaxNode): SyntaxNode | undefined {
  let current = node.parent;
  while (current !== null) {
    if (current.type === 'class_definition') return current;
    current = current.parent;
  }
  return undefined;
}

function ownerFunctionNodeFor(node: SyntaxNode): SyntaxNode | undefined {
  let current = node.parent;
  while (current !== null) {
    if (current.type === 'function_definition') return current;
    current = current.parent;
  }
  return undefined;
}

function ownerDefIdFor(node: SyntaxNode, filePath: string): string | undefined {
  const owner = ownerClassNodeFor(node);
  const nameNode = owner?.childForFieldName('name');
  if (owner === undefined || nameNode === null || nameNode === undefined) return undefined;
  return defId(filePath, owner, 'Class', nameNode.text);
}

function defId(filePath: string, node: SyntaxNode, type: 'Class', name: string): string {
  const range = capture('def', node).range;
  return `def:${filePath}#${range.startLine}:${range.startCol}:${type}:${name}`;
}

function firstParameterIdentifier(functionNode: SyntaxNode): SyntaxNode | undefined {
  const parameters = functionNode.childForFieldName('parameters');
  if (parameters === null) return undefined;
  return firstIdentifierChild(parameters);
}

function referenceNameNode(node: SyntaxNode): SyntaxNode | undefined {
  if (node.type === 'identifier') return node;
  if (node.type === 'attribute') return node.childForFieldName('attribute') ?? undefined;
  return firstIdentifierChild(node);
}

function isCallFunctionAttribute(node: SyntaxNode): boolean {
  const parent = node.parent;
  if (parent?.type !== 'call') return false;
  const fn = parent.childForFieldName('function');
  return fn !== null && isSameSyntaxNode(fn, node);
}

function memberAccessKind(node: SyntaxNode): 'read' | 'write' {
  const parent = node.parent;
  if (parent?.type !== 'assignment' && parent?.type !== 'augmented_assignment') return 'read';
  const left = parent.childForFieldName('left') ?? parent.namedChild(0);
  return left !== null && isSameSyntaxNode(left, node) ? 'write' : 'read';
}

function countArguments(args: SyntaxNode | null): number | undefined {
  if (args === null) return undefined;
  let count = 0;
  for (let index = 0; index < args.namedChildCount; index++) {
    const child = args.namedChild(index);
    if (child === null) continue;
    if (child.type === 'comment') continue;
    count++;
  }
  return count;
}

function firstNamedChildOfType(node: SyntaxNode, type: string): SyntaxNode | undefined {
  for (let index = 0; index < node.namedChildCount; index++) {
    const child = node.namedChild(index);
    if (child?.type === type) return child;
  }
  return undefined;
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

function firstIdentifierChild(node: SyntaxNode): SyntaxNode | undefined {
  for (let index = 0; index < node.namedChildCount; index++) {
    const child = node.namedChild(index);
    if (child?.type === 'identifier') return child;
    if (child !== null) {
      const nested = firstIdentifierChild(child);
      if (nested !== undefined) return nested;
    }
  }
  return undefined;
}

function lastIdentifierChild(node: SyntaxNode): SyntaxNode | undefined {
  let found: SyntaxNode | undefined;
  walk(node, (candidate) => {
    if (candidate.type === 'identifier') found = candidate;
  });
  return found;
}

function lastNamedChild(node: SyntaxNode): SyntaxNode | null {
  for (let index = node.namedChildCount - 1; index >= 0; index--) {
    const child = node.namedChild(index);
    if (child !== null) return child;
  }
  return null;
}

function moduleHead(value: string): string {
  return value.split('.').filter(Boolean)[0] ?? value;
}

function moduleTail(value: string): string {
  const parts = value.split('.').filter(Boolean);
  return parts[parts.length - 1] ?? value;
}

function isSameSyntaxNode(a: SyntaxNode, b: SyntaxNode): boolean {
  return a.type === b.type && a.startIndex === b.startIndex && a.endIndex === b.endIndex;
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
  for (let index = 0; index < node.namedChildCount; index++) {
    const child = node.namedChild(index);
    if (child !== null) walk(child, visit);
  }
}

function dedupeCaptureMatches(matches: readonly CaptureMatch[]): readonly CaptureMatch[] {
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
