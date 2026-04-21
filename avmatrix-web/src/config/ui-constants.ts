// Centralized UI defaults for the local runtime path plus a few legacy
// compatibility constants that still exist while older provider-based modules
// are being retired.
export const ERROR_RESET_DELAY_MS = 3000;
export const BACKEND_URL_DEBOUNCE_MS = 500;

export const DEFAULT_BACKEND_URL = 'http://localhost:4747';
export const DEFAULT_OLLAMA_BASE_URL = 'http://localhost:11434';
export const DEFAULT_OPENROUTER_BASE_URL = 'https://openrouter.ai/api/v1';

/** Minimum Node.js version required by the avmatrix CLI (injected by Vite from package.json engines). */
declare const __REQUIRED_NODE_VERSION__: string;
export const REQUIRED_NODE_VERSION = __REQUIRED_NODE_VERSION__;
