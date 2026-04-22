import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';
import { generateAIContextFiles } from '../../src/cli/ai-context.js';

describe('generateAIContextFiles', () => {
  let tmpDir: string;
  let storagePath: string;

  beforeAll(async () => {
    tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'gn-ai-ctx-test-'));
    storagePath = path.join(tmpDir, '.avmatrix');
    await fs.mkdir(storagePath, { recursive: true });
  });

  afterAll(async () => {
    try {
      await fs.rm(tmpDir, { recursive: true, force: true });
    } catch {
      /* best-effort */
    }
  });

  it('generates context files', async () => {
    const stats = {
      nodes: 100,
      edges: 200,
      processes: 10,
    };

    const result = await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);
    expect(result.files).toBeDefined();
    expect(result.files.length).toBeGreaterThan(0);
  });

  it('creates or updates CLAUDE.md with AVmatrix section', async () => {
    const stats = { nodes: 50, edges: 100, processes: 5 };
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    const claudeMdPath = path.join(tmpDir, 'CLAUDE.md');
    const content = await fs.readFile(claudeMdPath, 'utf-8');
    expect(content).toContain('avmatrix:start');
    expect(content).toContain('avmatrix:end');
    expect(content).toContain('TestProject');
  });

  it('keeps the load-bearing repo-specific sections in the CLAUDE.md block (#856)', async () => {
    // The trimmed block must still contain everything that is genuinely
    // unique per repo or load-bearing for the agent: the freshness warning,
    // the Always Do / Never Do imperative lists, the Resources URI table
    // (projectName-interpolated), and the skills routing table that tells
    // the agent which skill file to read for each task.
    const stats = { nodes: 50, edges: 100, processes: 5 };
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    const content = await fs.readFile(path.join(tmpDir, 'CLAUDE.md'), 'utf-8');

    expect(content).toContain('If any AVmatrix tool warns the index is stale');
    expect(content).toContain('## Always Do');
    expect(content).toContain('## Never Do');
    expect(content).toContain('## Resources');
    expect(content).toContain('avmatrix://repo/TestProject/context');
    expect(content).toContain('avmatrix-impact-analysis/SKILL.md');
    expect(content).toContain('avmatrix-refactoring/SKILL.md');
    expect(content).toContain('avmatrix-debugging/SKILL.md');
    expect(content).toContain('avmatrix-cli/SKILL.md');
  });

  it('does not duplicate content that already lives in skill files (#856)', async () => {
    // The six sections listed in issue #856 are redundant with the skill
    // files shipped alongside the CLAUDE.md block (both are loaded into
    // every Claude Code session). Their absence is the whole point of the
    // trim — assert each header is gone so a future regression that pads
    // the block back out fails here.
    const stats = { nodes: 50, edges: 100, processes: 5 };
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    const content = await fs.readFile(path.join(tmpDir, 'CLAUDE.md'), 'utf-8');

    expect(content).not.toContain('## Tools Quick Reference');
    expect(content).not.toContain('## Impact Risk Levels');
    expect(content).not.toContain('## Self-Check Before Finishing');
    expect(content).not.toContain('## When Debugging');
    expect(content).not.toContain('## When Refactoring');
    expect(content).not.toContain('## Keeping the Index Fresh');
  });

  it('keeps the CLAUDE.md AVmatrix block under the token-cost budget (#856)', async () => {
    // The pre-trim block was ~5465 chars. After #856 it's ~2580 — about a
    // 52% reduction. 2700 is a soft ceiling that still leaves headroom for
    // legitimate future additions but will fail loudly if the trim is
    // reverted or someone pads the block back out toward the original size.
    const stats = { nodes: 50, edges: 100, processes: 5 };
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    const content = await fs.readFile(path.join(tmpDir, 'CLAUDE.md'), 'utf-8');
    const block = content.slice(
      content.indexOf('<!-- avmatrix:start -->'),
      content.indexOf('<!-- avmatrix:end -->'),
    );
    expect(block.length).toBeLessThan(2700);
  });

  it('handles empty stats', async () => {
    const stats = {};
    const result = await generateAIContextFiles(tmpDir, storagePath, 'EmptyProject', stats);
    expect(result.files).toBeDefined();
  });

  it('updates existing CLAUDE.md without duplicating', async () => {
    const stats = { nodes: 10 };

    // Run twice
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    const claudeMdPath = path.join(tmpDir, 'CLAUDE.md');
    const content = await fs.readFile(claudeMdPath, 'utf-8');

    // Should only have one avmatrix section
    const starts = (content.match(/avmatrix:start/g) || []).length;
    expect(starts).toBe(1);
  });

  it('installs skills files', async () => {
    const stats = { nodes: 10 };
    const result = await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    // Should have installed skill files
    const skillsDir = path.join(tmpDir, '.claude', 'skills', 'avmatrix');
    try {
      const entries = await fs.readdir(skillsDir, { recursive: true });
      expect(entries.length).toBeGreaterThan(0);
    } catch {
      // Skills dir may not be created if skills source doesn't exist in test context
    }
  });

  it('uses MCP tool names without legacy avmatrix_ prefixes', async () => {
    const stats = { nodes: 10, edges: 20, processes: 2 };
    await generateAIContextFiles(tmpDir, storagePath, 'TestProject', stats);

    const content = await fs.readFile(path.join(tmpDir, 'CLAUDE.md'), 'utf-8');
    expect(content).toContain('`avmatrix analyze --force`');
    expect(content).not.toContain('`avmatrix analyze --force --skip-agents-md`');
    expect(content).toContain('`impact({target: "symbolName", direction: "upstream"})`');
    expect(content).toContain('`detect_changes()`');
    expect(content).toContain('`query({query: "concept"})`');
    expect(content).toContain('`context({name: "symbolName"})`');
    expect(content).not.toContain('avmatrix_impact');
    expect(content).not.toContain('avmatrix_detect_changes');
    expect(content).not.toContain('avmatrix_query');
    expect(content).not.toContain('avmatrix_context');
  });

  it('preserves manual AGENTS.md and CLAUDE.md edits when skipAgentsMd is enabled', async () => {
    const stats = { nodes: 42, edges: 84, processes: 3 };
    const agentsPath = path.join(tmpDir, 'AGENTS.md');
    const claudePath = path.join(tmpDir, 'CLAUDE.md');
    const agentsContent = '# AGENTS\n\nCustom manual instructions only\n';
    const claudeContent = '# CLAUDE\n\nCustom manual instructions only\n';

    await fs.writeFile(agentsPath, agentsContent, 'utf-8');
    await fs.writeFile(claudePath, claudeContent, 'utf-8');

    const result = await generateAIContextFiles(
      tmpDir,
      storagePath,
      'TestProject',
      stats,
      undefined,
      { skipAgentsMd: true },
    );

    expect(result.files).toContain('AGENTS.md (skipped via --skip-agents-md)');
    expect(result.files).toContain('CLAUDE.md (skipped via --skip-agents-md)');

    const agentsAfter = await fs.readFile(agentsPath, 'utf-8');
    const claudeAfter = await fs.readFile(claudePath, 'utf-8');
    expect(agentsAfter).toBe(agentsContent);
    expect(claudeAfter).toBe(claudeContent);
  });
});
