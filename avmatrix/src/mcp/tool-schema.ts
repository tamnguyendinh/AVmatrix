export interface JsonSchema {
  type?: 'object' | 'string' | 'number' | 'boolean' | 'array';
  description?: string;
  default?: unknown;
  items?: JsonSchema;
  enum?: readonly string[];
  properties?: Record<string, JsonSchema>;
  required?: string[];
  oneOf?: Array<Pick<JsonSchema, 'required'>>;
  anyOf?: Array<Pick<JsonSchema, 'required'>>;
  minLength?: number;
}

export interface ObjectSchema extends JsonSchema {
  type: 'object';
  properties: Record<string, JsonSchema>;
  required: string[];
}
