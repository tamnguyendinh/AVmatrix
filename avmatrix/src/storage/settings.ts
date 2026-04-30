import fs from 'fs/promises';
import path from 'path';
import { getStoragePath } from './repo-manager.js';

export interface AVmatrixSettings {
  maxExecutionFlows: number;
}

export const DEFAULT_MAX_EXECUTION_FLOWS = 700;

const DEFAULT_SETTINGS: AVmatrixSettings = {
  maxExecutionFlows: DEFAULT_MAX_EXECUTION_FLOWS,
};

export const getSettingsPath = (repoPath: string): string => {
  return path.join(getStoragePath(repoPath), 'settings.json');
};

const parsePositiveInteger = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isInteger(value) && value > 0) {
    return value;
  }
  return null;
};

const normalizeSettings = (value: unknown): AVmatrixSettings => {
  const parsed =
    typeof value === 'object' && value !== null ? (value as Partial<AVmatrixSettings>) : {};
  return {
    maxExecutionFlows:
      parsePositiveInteger(parsed.maxExecutionFlows) ?? DEFAULT_MAX_EXECUTION_FLOWS,
  };
};

export const loadSettings = async (repoPath: string): Promise<AVmatrixSettings> => {
  try {
    const raw = await fs.readFile(getSettingsPath(repoPath), 'utf-8');
    return normalizeSettings(JSON.parse(raw));
  } catch (error: unknown) {
    if ((error as NodeJS.ErrnoException | undefined)?.code === 'ENOENT') {
      await saveSettings(repoPath, DEFAULT_SETTINGS);
      return { ...DEFAULT_SETTINGS };
    }
    return { ...DEFAULT_SETTINGS };
  }
};

export const saveSettings = async (repoPath: string, settings: AVmatrixSettings): Promise<void> => {
  const dir = getStoragePath(repoPath);
  await fs.mkdir(dir, { recursive: true });
  await fs.writeFile(
    getSettingsPath(repoPath),
    JSON.stringify(normalizeSettings(settings), null, 2),
    'utf-8',
  );
};
