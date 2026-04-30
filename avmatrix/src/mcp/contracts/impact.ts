import type { ObjectSchema } from '../tool-schema.js';

export const IMPACT_ALLOWED_DIRECTIONS = ['upstream', 'downstream'] as const;
export type ImpactDirection = (typeof IMPACT_ALLOWED_DIRECTIONS)[number];

export const IMPACT_ALLOWED_RELATION_TYPES = [
  'CALLS',
  'IMPORTS',
  'EXTENDS',
  'IMPLEMENTS',
  'HAS_METHOD',
  'HAS_PROPERTY',
  'METHOD_OVERRIDES',
  'OVERRIDES',
  'METHOD_IMPLEMENTS',
  'ACCESSES',
  'HANDLES_ROUTE',
  'FETCHES',
  'HANDLES_TOOL',
  'ENTRY_POINT_OF',
  'WRAPS',
] as const;
export type ImpactRelationType = (typeof IMPACT_ALLOWED_RELATION_TYPES)[number];

export const IMPACT_DEFAULT_RELATION_TYPES = [
  'CALLS',
  'IMPORTS',
  'EXTENDS',
  'IMPLEMENTS',
  'METHOD_OVERRIDES',
  'OVERRIDES',
  'METHOD_IMPLEMENTS',
] as const satisfies readonly ImpactRelationType[];

export const IMPACT_DEFAULTS = {
  direction: 'upstream' as ImpactDirection,
  maxDepth: 3,
  includeTests: false,
  minConfidence: 0,
} as const;

const IMPACT_DIRECTION_SET = new Set<string>(IMPACT_ALLOWED_DIRECTIONS);
const IMPACT_RELATION_TYPE_SET = new Set<string>(IMPACT_ALLOWED_RELATION_TYPES);

export interface ImpactInputLike {
  target?: unknown;
  target_uid?: unknown;
  direction?: unknown;
  file_path?: unknown;
  kind?: unknown;
  maxDepth?: unknown;
  relationTypes?: unknown;
  includeTests?: unknown;
  minConfidence?: unknown;
  repo?: unknown;
}

export interface ParsedImpactInput {
  target?: string;
  target_uid?: string;
  direction: ImpactDirection;
  file_path?: string;
  kind?: string;
  maxDepth: number;
  relationTypes: ImpactRelationType[];
  includeTests: boolean;
  minConfidence: number;
  repo?: string;
}

export interface ImpactValidationError {
  error: string;
  field: string;
  allowedValues?: string[];
}

export type ImpactParseResult =
  | { ok: true; value: ParsedImpactInput }
  | { ok: false; error: ImpactValidationError };

function asOptionalTrimmedString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined;
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

function parseOptionalBoolean(
  value: unknown,
  defaultValue: boolean,
): boolean | ImpactValidationError {
  if (value === undefined) return defaultValue;
  if (typeof value === 'boolean') return value;
  if (value === 'true') return true;
  if (value === 'false') return false;
  return {
    error: 'Impact validation failed: includeTests must be a boolean.',
    field: 'includeTests',
  };
}

function parseOptionalNumber(
  value: unknown,
  options: { field: string; defaultValue: number; min: number; max?: number; integer?: boolean },
): number | ImpactValidationError {
  if (value === undefined) return options.defaultValue;

  const parsed =
    typeof value === 'number'
      ? value
      : typeof value === 'string' && value.trim().length > 0
        ? Number(value)
        : NaN;

  if (!Number.isFinite(parsed)) {
    return {
      error: `Impact validation failed: ${options.field} must be a valid number.`,
      field: options.field,
    };
  }

  if (options.integer && !Number.isInteger(parsed)) {
    return {
      error: `Impact validation failed: ${options.field} must be an integer.`,
      field: options.field,
    };
  }

  if (parsed < options.min || (options.max !== undefined && parsed > options.max)) {
    const range =
      options.max !== undefined ? `${options.min} and ${options.max}` : `at least ${options.min}`;
    return {
      error: `Impact validation failed: ${options.field} must be between ${range}.`,
      field: options.field,
    };
  }

  return parsed;
}

function parseDirection(value: unknown): ImpactDirection | ImpactValidationError {
  const direction = asOptionalTrimmedString(value);
  if (!direction || !IMPACT_DIRECTION_SET.has(direction)) {
    return {
      error: `Impact validation failed: direction must be one of ${IMPACT_ALLOWED_DIRECTIONS.join(', ')}.`,
      field: 'direction',
      allowedValues: [...IMPACT_ALLOWED_DIRECTIONS],
    };
  }
  return direction as ImpactDirection;
}

function normalizeRelationTypes(
  relationTypes: unknown,
): ImpactRelationType[] | ImpactValidationError {
  if (relationTypes === undefined) {
    return [...IMPACT_DEFAULT_RELATION_TYPES];
  }

  if (!Array.isArray(relationTypes)) {
    return {
      error: 'Impact validation failed: relationTypes must be an array of relation names.',
      field: 'relationTypes',
      allowedValues: [...IMPACT_ALLOWED_RELATION_TYPES],
    };
  }

  if (relationTypes.length === 0) {
    return {
      error:
        'Impact validation failed: relationTypes must include at least one valid relation when provided.',
      field: 'relationTypes',
      allowedValues: [...IMPACT_ALLOWED_RELATION_TYPES],
    };
  }

  const rawTypes = relationTypes.map((entry) =>
    typeof entry === 'string' ? entry.trim() : String(entry),
  );
  const invalid = rawTypes.filter((entry) => !IMPACT_RELATION_TYPE_SET.has(entry));
  if (invalid.length > 0) {
    return {
      error: `Impact validation failed: invalid relationTypes: ${invalid.join(', ')}.`,
      field: 'relationTypes',
      allowedValues: [...IMPACT_ALLOWED_RELATION_TYPES],
    };
  }

  const expanded = rawTypes.flatMap((entry) =>
    entry === 'OVERRIDES' ? (['OVERRIDES', 'METHOD_OVERRIDES'] as const) : [entry],
  );
  return [...new Set(expanded)] as ImpactRelationType[];
}

export function parseImpactInput(input: ImpactInputLike): ImpactParseResult {
  const target = asOptionalTrimmedString(input.target);
  const targetUid = asOptionalTrimmedString(input.target_uid);

  if (!target && !targetUid) {
    return {
      ok: false,
      error: {
        error: 'Impact validation failed: provide either target or target_uid.',
        field: 'target',
      },
    };
  }

  const direction = parseDirection(input.direction);
  if (typeof direction !== 'string') {
    return { ok: false, error: direction };
  }

  const maxDepth = parseOptionalNumber(input.maxDepth, {
    field: 'maxDepth',
    defaultValue: IMPACT_DEFAULTS.maxDepth,
    min: 1,
    integer: true,
  });
  if (typeof maxDepth !== 'number') {
    return { ok: false, error: maxDepth };
  }

  const minConfidence = parseOptionalNumber(input.minConfidence, {
    field: 'minConfidence',
    defaultValue: IMPACT_DEFAULTS.minConfidence,
    min: 0,
    max: 1,
  });
  if (typeof minConfidence !== 'number') {
    return { ok: false, error: minConfidence };
  }

  const includeTests = parseOptionalBoolean(input.includeTests, IMPACT_DEFAULTS.includeTests);
  if (typeof includeTests !== 'boolean') {
    return { ok: false, error: includeTests };
  }

  const relationTypes = normalizeRelationTypes(input.relationTypes);
  if (!Array.isArray(relationTypes)) {
    return { ok: false, error: relationTypes };
  }

  return {
    ok: true,
    value: {
      target,
      target_uid: targetUid,
      direction,
      file_path: asOptionalTrimmedString(input.file_path),
      kind: asOptionalTrimmedString(input.kind),
      maxDepth,
      relationTypes,
      includeTests,
      minConfidence,
      repo: asOptionalTrimmedString(input.repo),
    },
  };
}

export function buildImpactInputSchema(): ObjectSchema {
  return {
    type: 'object',
    properties: {
      target: {
        type: 'string',
        description: 'Name of function, class, or file to analyze',
        minLength: 1,
      },
      target_uid: {
        type: 'string',
        description:
          'Direct symbol UID from prior tool results (zero-ambiguity lookup, skips target resolution)',
        minLength: 1,
      },
      direction: {
        type: 'string',
        description: 'upstream (what depends on this) or downstream (what this depends on)',
        enum: IMPACT_ALLOWED_DIRECTIONS,
        default: IMPACT_DEFAULTS.direction,
      },
      file_path: {
        type: 'string',
        description: 'File path hint to disambiguate common names',
      },
      kind: {
        type: 'string',
        description:
          "Kind filter to disambiguate common names (e.g. 'Function', 'Class', 'Method', 'Interface', 'Constructor')",
      },
      maxDepth: {
        type: 'number',
        description: `Max relationship depth (default: ${IMPACT_DEFAULTS.maxDepth})`,
        default: IMPACT_DEFAULTS.maxDepth,
      },
      relationTypes: {
        type: 'array',
        items: { type: 'string', enum: IMPACT_ALLOWED_RELATION_TYPES },
        description: `Filter: ${IMPACT_ALLOWED_RELATION_TYPES.join(', ')} (default: ${IMPACT_DEFAULT_RELATION_TYPES.join(', ')})`,
      },
      includeTests: {
        type: 'boolean',
        description: `Include test files (default: ${IMPACT_DEFAULTS.includeTests})`,
        default: IMPACT_DEFAULTS.includeTests,
      },
      minConfidence: {
        type: 'number',
        description: `Minimum confidence 0-1 (default: ${IMPACT_DEFAULTS.minConfidence})`,
        default: IMPACT_DEFAULTS.minConfidence,
      },
      repo: {
        type: 'string',
        description: 'Repository name or path. Omit if only one repo is indexed.',
      },
    },
    required: ['direction'],
    oneOf: [{ required: ['target'] }, { required: ['target_uid'] }],
  };
}
