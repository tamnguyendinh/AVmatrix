/**
 * Compatibility wrapper for the legacy wiki command module.
 *
 * Remote wiki generation is disabled in local-only mode. The old import path
 * now resolves to the same local capability gate used by the public CLI.
 */

import { wikiGatedCommand } from './wiki-gated.js';

export interface WikiCommandOptions {}

export const wikiCommand = async (_inputPath?: string, _options?: WikiCommandOptions) => {
  await wikiGatedCommand();
};
