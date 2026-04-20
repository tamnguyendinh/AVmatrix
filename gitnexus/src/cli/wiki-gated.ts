import { loadRuntimeConfig, saveRuntimeConfig, type WikiMode } from '../storage/runtime-config.js';

const WIKI_MODE_VALUES: WikiMode[] = ['off', 'local'];

export const formatWikiModeStatus = (mode: WikiMode): string => {
  if (mode === 'local') {
    return [
      '',
      '  Wiki capability mode: local',
      '',
      '  Local wiki mode is reserved, but the local wiki engine is not available yet in this build.',
      '  GitNexus will not fall back to any remote wiki service.',
      '',
    ].join('\n');
  }

  return [
    '',
    '  Wiki capability mode: off',
    '',
    '  Wiki generation is disabled in local-only mode.',
    '  Run `gitnexus wiki-mode local` later when the local wiki engine is ready.',
    '',
  ].join('\n');
};

export const wikiGatedCommand = async (): Promise<void> => {
  const config = await loadRuntimeConfig();
  console.log(formatWikiModeStatus(config.wikiMode));
  process.exitCode = 1;
};

export const wikiModeCommand = async (mode?: string): Promise<void> => {
  if (!mode) {
    const config = await loadRuntimeConfig();
    console.log(formatWikiModeStatus(config.wikiMode));
    return;
  }

  const normalized = mode.trim().toLowerCase();
  if (!WIKI_MODE_VALUES.includes(normalized as WikiMode)) {
    console.error('\n  Invalid wiki mode. Use `off` or `local`.\n');
    process.exitCode = 1;
    return;
  }

  const nextMode = normalized as WikiMode;
  await saveRuntimeConfig({ wikiMode: nextMode });
  console.log(formatWikiModeStatus(nextMode));
};

