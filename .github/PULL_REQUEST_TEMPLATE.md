## Summary

<!-- One or two sentences: what does this PR change? -->

## Motivation / context

<!-- Why is this change needed? Link issues, ADRs, or prior discussion. -->

## Areas touched

<!-- Check all that apply -->

- [ ] `avmatrix/` (CLI / core / MCP server)
- [ ] `avmatrix-web/` (Vite / React UI)
- [ ] `.github/` (workflows, actions)
- [ ] Other tooling / repository infrastructure
- [ ] Docs / agent config only (`AGENTS.md`, `CLAUDE.md`, `.cursor/`, `llms.txt`, etc.)

## Scope & constraints

**In scope**

- <!-- bullets -->

**Explicitly out of scope / not done here**

- <!-- bullets — prevents reviewers assuming missing work is an oversight -->

## Implementation notes

<!-- Optional: design choices, tradeoffs, follow-ups -->

## Testing & verification

<!-- What you ran; paste commands. Omit sections that do not apply. -->

- [ ] `cd avmatrix && npm test`
- [ ] `cd avmatrix && npm run test:integration` *(if core/indexing/MCP paths changed)*
- [ ] `cd avmatrix && npx tsc --noEmit`
- [ ] `cd avmatrix-web && npm test` *(if web changed)*
- [ ] `cd avmatrix-web && npx tsc -b --noEmit` *(if web changed)*
- [ ] Manual / Playwright E2E *(note environment — see `avmatrix-web/e2e/`)*

## Risk & rollout

<!-- Breaking changes, migrations, index refresh (`npx avmatrix analyze`), release notes -->

## Checklist

- [ ] PR body meets repo minimum length (workflow may label short descriptions)
- [ ] If `AGENTS.md` / overlays changed: headers, scope block, and changelog updated per project conventions
- [ ] No secrets, tokens, or machine-specific paths committed
