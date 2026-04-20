import fs from 'fs/promises';
import path from 'path';
import { getGlobalDir } from './repo-manager.js';

export type WikiMode = 'off' | 'local';

export interface RuntimeConfig {
  wikiMode: WikiMode;
}

const DEFAULT_RUNTIME_CONFIG: RuntimeConfig = {
  wikiMode: 'off',
};

export const getRuntimeConfigPath = (): string => {
  return path.join(getGlobalDir(), 'runtime.json');
};

export const loadRuntimeConfig = async (): Promise<RuntimeConfig> => {
  try {
    const raw = await fs.readFile(getRuntimeConfigPath(), 'utf-8');
    const parsed = JSON.parse(raw) as Partial<RuntimeConfig>;
    return {
      wikiMode: parsed.wikiMode === 'local' ? 'local' : 'off',
    };
  } catch {
    return { ...DEFAULT_RUNTIME_CONFIG };
  }
};

export const saveRuntimeConfig = async (config: RuntimeConfig): Promise<void> => {
  const dir = getGlobalDir();
  await fs.mkdir(dir, { recursive: true });
  await fs.writeFile(getRuntimeConfigPath(), JSON.stringify(config, null, 2), 'utf-8');
};

