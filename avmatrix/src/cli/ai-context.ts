/**
 * AI Context Generator
 *
 * Creates AGENTS.md and CLAUDE.md with full inline AVmatrix context.
 * AGENTS.md is the standard read by Cursor, Windsurf, OpenCode, Codex, Cline, etc.
 * CLAUDE.md is for Claude Code which only reads that file.
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';
import { type GeneratedSkillInfo } from './skill-gen.js';

// ESM equivalent of __dirname
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

interface RepoStats {
  files?: number;
  nodes?: number;
  edges?: number;
  communities?: number;
  clusters?: number; // Aggregated cluster count (what tools show)
  processes?: number;
}

export interface AIContextOptions {
  skipAgentsMd?: boolean;
  noStats?: boolean;
}

const AVMATRIX_START_MARKER = '<!-- avmatrix:start -->';
const AVMATRIX_END_MARKER = '<!-- avmatrix:end -->';
const MANAGED_SECTION_PATTERN =
  /<!--\s*([a-z0-9-]+):start\s*-->[\s\S]*?#\s+[^\n]*Code Intelligence[\s\S]*?<!--\s*\1:end\s*-->/i;

/**
 * Generate the full AVmatrix context content.
 *
 * Design principles (learned from real agent behavior and industry research):
 * - Inline critical workflows — skills are skipped 56% of the time (Vercel eval data)
 * - Use RFC 2119 language (MUST, NEVER, ALWAYS) — models follow imperative rules
 * - Three-tier boundaries (Always/When/Never) — proven to change model behavior
 * - Keep under 120 lines — adherence degrades past 150 lines
 * - Exact tool commands with parameters — vague directives get ignored
 * - Self-review checklist — forces model to verify its own work
 */
async function findGroupsContainingRegistryName(registryName: string): Promise<string[]> {
  const { listGroups, getDefaultAVmatrixDir, getGroupDir } =
    await import('../core/group/storage.js');
  const { loadGroupConfig } = await import('../core/group/config-parser.js');
  const names = await listGroups();
  const hits: string[] = [];
  for (const g of names) {
    try {
      const config = await loadGroupConfig(getGroupDir(getDefaultAVmatrixDir(), g));
      if (Object.values(config.repos).some((r) => r === registryName)) hits.push(config.name);
    } catch {
      // skip invalid or unreadable groups
    }
  }
  return hits;
}

function generateAVmatrixContent(
  projectName: string,
  stats: RepoStats,
  generatedSkills?: GeneratedSkillInfo[],
  groupNames?: string[],
  noStats?: boolean,
): string {
  const generatedRows =
    generatedSkills && generatedSkills.length > 0
      ? generatedSkills
          .map(
            (s) =>
              `| Work in the ${s.label} area (${s.symbolCount} symbols) | \`.claude/skills/generated/${s.name}/SKILL.md\` |`,
          )
          .join('\n')
      : '';

  const skillsTable = `| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | \`.claude/skills/avmatrix/avmatrix-exploring/SKILL.md\` |
| Blast radius / "What breaks if I change X?" | \`.claude/skills/avmatrix/avmatrix-impact-analysis/SKILL.md\` |
| Trace bugs / "Why is X failing?" | \`.claude/skills/avmatrix/avmatrix-debugging/SKILL.md\` |
| Rename / extract / split / refactor | \`.claude/skills/avmatrix/avmatrix-refactoring/SKILL.md\` |
| Tools, resources, schema reference | \`.claude/skills/avmatrix/avmatrix-guide/SKILL.md\` |
| Index, status, clean, and wiki capability CLI commands | \`.claude/skills/avmatrix/avmatrix-cli/SKILL.md\` |${generatedRows ? '\n' + generatedRows : ''}`;

  return `${AVMATRIX_START_MARKER}
# AVmatrix — Code Intelligence

This project is indexed by AVmatrix as **${projectName}**${noStats ? '' : ` (${stats.nodes || 0} symbols, ${stats.edges || 0} relationships, ${stats.processes || 0} execution flows)`}. Use the AVmatrix MCP tools to understand code, assess impact, and navigate safely.

> If any AVmatrix tool warns the index is stale, run \`avmatrix analyze\` in terminal first.

## Always Do

- **MUST refresh the graph before any graph-based work.** From the repo root, run \`avmatrix analyze --force --skip-agents-md\` before using \`query\`, \`context\`, \`impact\`, \`detect_changes\`, \`rename\`, or \`cypher\`.
- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run \`impact({target: "symbolName", direction: "upstream"})\` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run \`detect_changes()\` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use \`query({query: "concept"})\` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use \`context({name: "symbolName"})\`.

## Never Do

- NEVER edit a function, class, or method without first running \`impact\` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use \`rename\` which understands the call graph.
- NEVER commit changes without running \`detect_changes()\` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| \`avmatrix://repo/${projectName}/context\` | Codebase overview, check index freshness |
| \`avmatrix://repo/${projectName}/clusters\` | All functional areas |
| \`avmatrix://repo/${projectName}/processes\` | All execution flows |
| \`avmatrix://repo/${projectName}/process/{name}\` | Step-by-step execution trace |

${
  groupNames && groupNames.length > 0
    ? `## Cross-Repo Groups

This repository is listed under AVmatrix **group(s): ${groupNames.join(', ')}**. For blast radius across repository boundaries, use MCP tools \`group_impact\`, \`group_sync\`, \`group_query\`, \`group_contracts\`, \`group_status\`, and \`group_list\`. From the terminal: \`avmatrix group list\`, \`avmatrix group sync <name>\`, \`avmatrix group impact <name> --target <symbol> --repo <group-path>\`.

`
    : ''
}## CLI

${skillsTable}

${AVMATRIX_END_MARKER}`;
}

/**
 * Check if a file exists
 */
async function fileExists(filePath: string): Promise<boolean> {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

/**
 * Create or update the AVmatrix section in a file.
 * - If the file doesn't exist: create it with AVmatrix content
 * - If the file exists without any managed section: append the AVmatrix section
 * - If the file exists with a managed AVmatrix-era section from any prior rollout step:
 *   replace that section in place
 */
async function upsertAVmatrixSection(
  filePath: string,
  content: string,
): Promise<'created' | 'updated' | 'appended'> {
  const exists = await fileExists(filePath);

  if (!exists) {
    await fs.writeFile(filePath, content, 'utf-8');
    return 'created';
  }

  const existingContent = await fs.readFile(filePath, 'utf-8');

  // Replace whichever managed Code Intelligence block is already present.
  const existingSection = existingContent.match(MANAGED_SECTION_PATTERN);
  if (existingSection) {
    const newContent = existingContent.replace(MANAGED_SECTION_PATTERN, content);
    await fs.writeFile(filePath, newContent.trim() + '\n', 'utf-8');
    return 'updated';
  }

  // Append new section
  const newContent = existingContent.trim() + '\n\n' + content + '\n';
  await fs.writeFile(filePath, newContent, 'utf-8');
  return 'appended';
}

/**
 * Install AVmatrix skills to .claude/skills/avmatrix/
 * Works natively with Claude Code, Cursor, and GitHub Copilot
 */
async function installSkills(repoPath: string): Promise<string[]> {
  const skillsDir = path.join(repoPath, '.claude', 'skills', 'avmatrix');
  const legacySkillsDir = path.join(repoPath, '.claude', 'skills', 'avmatrix');
  const installedSkills: string[] = [];

  // Skill definitions bundled with the package
  const skills = [
    {
      name: 'avmatrix-exploring',
      outputName: 'avmatrix-exploring',
      description:
        'Use when the user asks how code works, wants to understand architecture, trace execution flows, or explore unfamiliar parts of the codebase. Examples: "How does X work?", "What calls this function?", "Show me the auth flow"',
    },
    {
      name: 'avmatrix-debugging',
      outputName: 'avmatrix-debugging',
      description:
        'Use when the user is debugging a bug, tracing an error, or asking why something fails. Examples: "Why is X failing?", "Where does this error come from?", "Trace this bug"',
    },
    {
      name: 'avmatrix-impact-analysis',
      outputName: 'avmatrix-impact-analysis',
      description:
        'Use when the user wants to know what will break if they change something, or needs safety analysis before editing code. Examples: "Is it safe to change X?", "What depends on this?", "What will break?"',
    },
    {
      name: 'avmatrix-refactoring',
      outputName: 'avmatrix-refactoring',
      description:
        'Use when the user wants to rename, extract, split, move, or restructure code safely. Examples: "Rename this function", "Extract this into a module", "Refactor this class", "Move this to a separate file"',
    },
    {
      name: 'avmatrix-guide',
      outputName: 'avmatrix-guide',
      description:
        'Use when the user asks about AVmatrix itself — available tools, how to query the knowledge graph, MCP resources, graph schema, or workflow reference. Examples: "What AVmatrix tools are available?", "How do I use AVmatrix?"',
    },
    {
      name: 'avmatrix-cli',
      outputName: 'avmatrix-cli',
      description:
        'Use when the user needs to run AVmatrix CLI commands like analyze/index a repo, check status, clean the index, inspect wiki capability mode, or list indexed repos. Examples: "Index this repo", "Reanalyze the codebase", "Check wiki mode"',
    },
  ];

  await fs.rm(legacySkillsDir, { recursive: true, force: true });

  for (const skill of skills) {
    const skillDir = path.join(skillsDir, skill.outputName);
    const skillPath = path.join(skillDir, 'SKILL.md');

    try {
      // Create skill directory
      await fs.mkdir(skillDir, { recursive: true });

      // Try to read from package skills directory
      const packageSkillPath = path.join(__dirname, '..', '..', 'skills', `${skill.name}.md`);
      let skillContent: string;

      try {
        skillContent = await fs.readFile(packageSkillPath, 'utf-8');
      } catch {
        // Fallback: generate minimal skill content
        skillContent = `---
name: ${skill.outputName}
description: ${skill.description}
---

# ${skill.outputName.charAt(0).toUpperCase() + skill.outputName.slice(1)}

${skill.description}

Use AVmatrix tools to accomplish this task.
`;
      }

      await fs.writeFile(skillPath, skillContent, 'utf-8');
      installedSkills.push(skill.outputName);
    } catch (err) {
      // Skip on error, don't fail the whole process
      console.warn(`Warning: Could not install skill ${skill.name}:`, err);
    }
  }

  return installedSkills;
}

/**
 * Generate AI context files after indexing
 */
export async function generateAIContextFiles(
  repoPath: string,
  _storagePath: string,
  projectName: string,
  stats: RepoStats,
  generatedSkills?: GeneratedSkillInfo[],
  options?: AIContextOptions,
): Promise<{ files: string[] }> {
  const groupNames = await findGroupsContainingRegistryName(projectName);
  const content = generateAVmatrixContent(
    projectName,
    stats,
    generatedSkills,
    groupNames,
    options?.noStats,
  );
  const createdFiles: string[] = [];

  if (!options?.skipAgentsMd) {
    // Create AGENTS.md (standard for Cursor, Windsurf, OpenCode, Cline, etc.)
    const agentsPath = path.join(repoPath, 'AGENTS.md');
    const agentsResult = await upsertAVmatrixSection(agentsPath, content);
    createdFiles.push(`AGENTS.md (${agentsResult})`);

    // Create CLAUDE.md (for Claude Code)
    const claudePath = path.join(repoPath, 'CLAUDE.md');
    const claudeResult = await upsertAVmatrixSection(claudePath, content);
    createdFiles.push(`CLAUDE.md (${claudeResult})`);
  } else {
    createdFiles.push('AGENTS.md (skipped via --skip-agents-md)');
    createdFiles.push('CLAUDE.md (skipped via --skip-agents-md)');
  }

  // Install skills to .claude/skills/avmatrix/
  const installedSkills = await installSkills(repoPath);
  if (installedSkills.length > 0) {
    createdFiles.push(`.claude/skills/avmatrix/ (${installedSkills.length} skills)`);
  }

  return { files: createdFiles };
}
