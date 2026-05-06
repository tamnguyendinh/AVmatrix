/**
 * Bridge between a language provider's `emitScopeCaptures` hook and the
 * `ScopeExtractor` (RFC #909 Ring 2 PKG #920).
 *
 * Extracted into its own module so it can be imported by test code
 * without pulling in `parse-worker.ts` — which has a top-level
 * `parentPort!.on('message', ...)` call that assumes a worker-thread
 * context and throws on direct import.
 *
 * The bridge:
 *
 *   1. Short-circuits when the provider has NOT implemented a scope-capture
 *      hook. Returns `undefined`; zero work done. This is the state of every
 *      language today — `ParsedFile` production stays dormant until a
 *      language migrates.
 *   2. Invokes the AST-aware hook when a worker root node is available,
 *      falling back to the source-text hook only for compatibility, then
 *      feeds the captures to `ScopeExtractor.extract`.
 *   3. **Swallows exceptions from either side.** A failure here returns
 *      `undefined` and emits a warning via `onWarn`; legacy parsing on
 *      the same file continues unaffected by the scope-extraction miss.
 *      Scope-based resolution is the new path under construction — it
 *      must not destabilize the legacy DAG.
 */

import type { ParsedFile, SupportedLanguages } from 'avmatrix-shared';
import { extract as extractScope } from './scope-extractor.js';
import type { LanguageProvider } from './language-provider.js';
import type { SyntaxNode } from './utils/ast-helpers.js';

/** Callback used to report scope-extraction warnings to the host (worker or direct). */
export type ScopeBridgeWarn = (message: string) => void;

export type ScopeExtractionMode = 'ast-reused' | 'compatibility-source' | 'no-hook' | 'failed';

export interface ScopeExtractionBridgeResult {
  readonly parsedFile?: ParsedFile;
  readonly mode: ScopeExtractionMode;
}

/**
 * Produce a `ParsedFile` for the given file, or `undefined` when the
 * provider hasn't migrated / the extractor throws. Never propagates
 * exceptions.
 */
export function extractParsedFile(
  provider: LanguageProvider,
  sourceText: string,
  filePath: string,
  onWarn?: ScopeBridgeWarn,
): ParsedFile | undefined {
  return extractParsedFileWithStats(provider, sourceText, filePath, undefined, undefined, onWarn)
    .parsedFile;
}

/**
 * AST-aware bridge used by parse workers. Prefer this over `extractParsedFile`
 * when the worker already has a tree-sitter root node for the file.
 */
export function extractParsedFileWithStats(
  provider: LanguageProvider,
  sourceText: string,
  filePath: string,
  language: SupportedLanguages | undefined,
  rootNode: SyntaxNode | undefined,
  onWarn?: ScopeBridgeWarn,
): ScopeExtractionBridgeResult {
  const canUseAst = provider.emitScopeCapturesFromTree !== undefined && rootNode !== undefined;
  if (!canUseAst && provider.emitScopeCaptures === undefined) return { mode: 'no-hook' };

  const mode: ScopeExtractionMode = canUseAst ? 'ast-reused' : 'compatibility-source';
  try {
    const captures = canUseAst
      ? provider.emitScopeCapturesFromTree!({
          sourceText,
          filePath,
          language: language ?? provider.id,
          rootNode,
        })
      : provider.emitScopeCaptures!(sourceText, filePath);
    return { parsedFile: extractScope(captures, filePath, provider), mode };
  } catch (err) {
    const message = `scope extraction failed for ${filePath}: ${
      err instanceof Error ? err.message : String(err)
    }`;
    if (onWarn !== undefined) onWarn(message);
    else console.warn(message);
    return { mode: 'failed' };
  }
}
