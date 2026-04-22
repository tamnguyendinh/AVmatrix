/**
 * MCP Command
 *
 * Starts the MCP server in standalone mode.
 * Loads all indexed repos from the global registry.
 * No longer depends on cwd — works from any directory.
 */

import { startMCPServer } from '../mcp/server.js';
import { LocalBackend } from '../mcp/local/local-backend.js';

const STARTUP_LOG_PREFIX = 'AVmatrix MCP [startup]';

function logStartup(stage: string, startedAt: number, details?: string): void {
  const elapsedMs = Date.now() - startedAt;
  const suffix = details ? ` ${details}` : '';
  process.stderr.write(`${STARTUP_LOG_PREFIX} stage=${stage} elapsedMs=${elapsedMs}${suffix}\n`);
}

export const mcpCommand = async () => {
  const startedAt = Date.now();

  // Prevent unhandled errors from crashing the MCP server process.
  // LadybugDB lock conflicts and transient errors should degrade gracefully.
  process.on('uncaughtException', (err) => {
    console.error(`AVmatrix MCP: uncaught exception — ${err.message}`);
    // Process is in an undefined state after uncaughtException — exit after flushing
    setTimeout(() => process.exit(1), 100);
  });
  process.on('unhandledRejection', (reason) => {
    const msg = reason instanceof Error ? reason.message : String(reason);
    console.error(`AVmatrix MCP: unhandled rejection — ${msg}`);
  });

  // Construct the backend, but do not block MCP handshake on repo discovery.
  // Repo refresh and DB work happen lazily on the first repo-aware request.
  const backend = new LocalBackend();
  logStartup('backend_created', startedAt);

  // Start MCP server immediately so the client can handshake before any
  // registry refresh or repo warm-up work runs.
  await startMCPServer(backend);
  logStartup('transport_connected', startedAt);
};
