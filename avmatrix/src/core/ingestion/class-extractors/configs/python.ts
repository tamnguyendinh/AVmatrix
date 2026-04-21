// avmatrix/src/core/ingestion/class-extractors/configs/python.ts

import { SupportedLanguages } from 'avmatrix-shared';
import type { ClassExtractionConfig } from '../../class-types.js';

export const pythonClassConfig: ClassExtractionConfig = {
  language: SupportedLanguages.Python,
  typeDeclarationNodes: ['class_definition'],
  ancestorScopeNodeTypes: ['class_definition'],
};
