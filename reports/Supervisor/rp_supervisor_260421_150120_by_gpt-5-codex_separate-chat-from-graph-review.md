# Separate Chat From Graph Review

- Plan: `docs/plans/2026-04-21-separate-chat-from-graph.md`
- Reviewed batch: `9fbff1f` (`refactor: separate chat runtime from graph state`)
- Verdict: `NOT APPROVED`
- Reviewer: `gpt-5-codex`
- Timestamp: `2026-04-21 15:01:20 +07:00`

## Primary finding

1. `HIGH` Missing required behavioral coverage for the new grounding-click active path.

The refactor moved the active transcript click chain onto the new chat surface:

- `avmatrix-web/src/components/ChatPanel.tsx:37`
- `avmatrix-web/src/components/ChatPanel.tsx:50`
- `avmatrix-web/src/components/right-panel/ChatTranscript.tsx:134`
- `avmatrix-web/src/components/right-panel/ChatTranscript.tsx:149`
- `avmatrix-web/src/components/MarkdownRenderer.tsx:93`
- `avmatrix-web/src/components/MarkdownRenderer.tsx:125`
- `avmatrix-web/src/hooks/useAppState.local-runtime.tsx:548`

That is now the real active wiring for:

- rendering `[[file refs]]` / `[[node refs]]`
- clicking a grounding link in transcript
- routing back through `handleTranscriptLinkClick`
- adding the code reference into app state

But the new tests only lock:

- analyze CTA and typing behavior in `avmatrix-web/test/unit/ChatPanel.test.tsx:69`
- bridge parsing via direct calls in `avmatrix-web/test/unit/useAppState.local-runtime.test.tsx:83`
- runtime lazy start / repo reset in `avmatrix-web/test/unit/ChatRuntimeContext.test.tsx:54`

I did not find any test that exercises the active `ChatPanel -> ChatTranscript -> MarkdownRenderer -> handleTranscriptLinkClick -> addCodeReference` click path after the refactor. The plan explicitly lists this as a mandatory behavioral check under:

- `Grounding vẫn hoạt động`
- `click grounding vẫn add code reference đúng`

Under the supervisor hard rule, a missing/stale test in the same refactor scope is a blocker, so I cannot approve this batch yet.

## What is already good

- `RightPanel` is now a shell that mounts `ChatPanel` / `ProcessesPanel` only.
  - `avmatrix-web/src/components/RightPanel.tsx:12`
  - `avmatrix-web/src/components/RightPanel.tsx:71`
- `ChatPanel` no longer reads graph state directly.
  - no `GraphCanvas`
  - no `useSigma`
  - no direct `useAppState()` in `ChatPanel`
- chat runtime moved into the new provider.
  - `avmatrix-web/src/hooks/chat-runtime/ChatRuntimeContext.tsx:33`
- `AppContent` now gets `refreshLLMSettings` from `useChatRuntime`, not old app context.
  - `avmatrix-web/src/App.tsx:61`
- targeted plan validation passes
- full `avmatrix-web` suite passes

## Validation run

- `cd avmatrix-web && npx vitest run test/unit/ChatRuntimeContext.test.tsx test/unit/ChatPanel.test.tsx test/unit/RightPanel.local-runtime.test.tsx test/unit/ChatComposer.test.tsx`
- `cd avmatrix-web && npx tsc -b --noEmit`
- `cd avmatrix-web && npm test`

Results:

- targeted plan tests: pass
- typecheck: pass
- full web suite: pass (`34/34` files, `280/280` tests)

## Scope note

Current worktree also has unrelated edits outside this review scope:

- `AGENTS.md`
- `CLAUDE.md`

They were not used in this verdict.
