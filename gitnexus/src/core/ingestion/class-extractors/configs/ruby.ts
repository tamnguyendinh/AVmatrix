// gitnexus/src/core/ingestion/class-extractors/configs/ruby.ts

import { SupportedLanguages } from 'avmatrix-shared';
import type { ClassExtractionConfig } from '../../class-types.js';

export const rubyClassConfig: ClassExtractionConfig = {
  language: SupportedLanguages.Ruby,
  typeDeclarationNodes: ['class'],
  ancestorScopeNodeTypes: ['module', 'class'],
};
