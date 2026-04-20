import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(testDir, '../..');
const cliEntry = path.join(repoRoot, 'src/cli/index.ts');

function runHelp(command: string) {
  return spawnSync(process.execPath, ['--import', 'tsx', cliEntry, command, '--help'], {
    cwd: repoRoot,
    encoding: 'utf8',
  });
}

function runRootHelp() {
  return spawnSync(process.execPath, ['--import', 'tsx', cliEntry, '--help'], {
    cwd: repoRoot,
    encoding: 'utf8',
  });
}

function runServeHelp() {
  return spawnSync(process.execPath, ['--import', 'tsx', cliEntry, 'serve', '--help'], {
    cwd: repoRoot,
    encoding: 'utf8',
  });
}

describe('CLI help surface', () => {
  it('query help keeps advanced search options without importing analyze deps', () => {
    const result = runHelp('query');

    expect(result.status).toBe(0);
    expect(result.stdout).toContain('--context <text>');
    expect(result.stdout).toContain('--goal <text>');
    expect(result.stdout).toContain('--content');
    expect(result.stderr).not.toContain('tree-sitter-kotlin');
  });

  it('context help keeps optional name and disambiguation flags', () => {
    const result = runHelp('context');

    expect(result.status).toBe(0);
    expect(result.stdout).toContain('context [options] [name]');
    expect(result.stdout).toContain('--uid <uid>');
    expect(result.stdout).toContain('--file <path>');
  });

  it('impact help keeps repo and include-tests flags', () => {
    const result = runHelp('impact');

    expect(result.status).toBe(0);
    expect(result.stdout).toContain('--depth <n>');
    expect(result.stdout).toContain('--include-tests');
    expect(result.stdout).toContain('--repo <name>');
  });

  it('wiki help shows provider, review, and verbose flags', () => {
    const result = runHelp('wiki');

    expect(result.status).toBe(0);
    expect(result.stdout).toContain('--provider <provider>');
    expect(result.stdout).toContain('--review');
    expect(result.stdout).toContain('-v, --verbose');
    expect(result.stdout).toContain('--model <model>');
    expect(result.stdout).toContain('--gist');
  });

  it('root help describes serve and mcp as local runtime surfaces', () => {
    const result = runRootHelp();

    expect(result.status).toBe(0);
    expect(result.stdout).toContain('serve [options]');
    expect(result.stdout).toContain('local HTTP bridge for the web UI');
    expect(result.stdout).toContain('shared session runtime');
    expect(result.stdout).toContain('mcp');
    expect(result.stdout).toContain('backed by the same');
    expect(result.stdout).toContain('local runtime core');
    expect(result.stdout).toContain('wiki-mode [mode]');
  });

  it('serve help enforces loopback-only host wording', () => {
    const result = runServeHelp();

    expect(result.status).toBe(0);
    expect(result.stdout).toContain('loopback only');
    expect(result.stdout).not.toContain('remote access');
  });
});
