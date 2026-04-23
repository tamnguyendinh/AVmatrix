# Web UI Plan: Straight Edges And Thinner Selected-Context Links

Date: 2026-04-23
Scope: `avmatrix-web/`
Status: Planned

## Goal

Reduce graph visual fatigue by:

1. switching graph links from curved edges to straight edges
2. reducing selected-node contextual edge thickness to roughly 70% of the current visual weight when ambient graph links are turned off

This plan is intentionally limited to frontend rendering. It does not change graph data, backend payloads, or graph semantics.

## Current Problems

1. Graph links are rendered as curves, which increases visual noise when many relationships overlap.
2. In the `all graph links off + selected node` mode, the contextual links still feel visually heavy.
3. A blanket global shrink of every edge state would be wrong, because the full-graph ambient mode should keep its current visual weight.

## What The Code Does Today

1. Sigma defaults to curved edges in `avmatrix-web/src/hooks/useSigma.ts`.
2. The graph adapter also stamps edges as `type: 'curved'` with a per-edge `curvature` in `avmatrix-web/src/lib/graph-adapter.ts`.
3. Base edge thickness is created in `graph-adapter.ts`, then further amplified in `useSigma.ts` for selected/highlight states.
4. The selected-node contextual edge styling currently shares too much of the same “strong edge” treatment, which is too heavy for the focused single-node mode.

## Target Behavior

1. Ambient graph links render as straight lines, not curves.
2. Ambient graph links keep their current color semantics, visibility/filter semantics, and current thickness.
3. Selected-node contextual links render at about 70% of their current thickness only when ambient graph links are off.
4. Selected-node contextual links keep their current emphasis when ambient graph links are on.
5. AI/query/blast-radius highlight edges keep their current emphasis unless a later pass explicitly changes them.
6. No backend or graph-model changes are introduced.

## Non-Goals

1. Do not redesign edge colors.
2. Do not change which relationship types exist or how they are filtered.
3. Do not change selected-node edge semantics, AI highlight semantics, or graph-links toggle semantics.
4. Do not add new user settings in this pass.
5. Do not change ambient edge thickness in any mode.
6. Do not change highlight-edge thickness in ambient-on mode.

## Design Rules

1. Treat this as a rendering adjustment, not a data-model change.
2. Keep the current edge-state semantics intact:
   - ambient graph links stay as they are today
   - selected-context links stay contextual-only
   - highlight-driven links keep their current visibility semantics
3. Apply the width reduction only to the `ambient off + selected-context on` rendering path.
4. Do not globally thin all edges at the source if the requested change is only for the selected-context mode.
5. Because this is a conditional render-state change, implement the width reduction in render policy logic, not in global source edge sizing from `graph-adapter.ts`.

## Refactor Safety Rule

If a change remains a narrow rendering adjustment, direct edits are acceptable.

If the edge rendering logic starts to grow into a larger rendering policy refactor, create a new helper/module in parallel first, wire it in, verify behavior, then remove or simplify the older path.

## Implementation Plan

### Workstream A — Switch Edge Geometry To Straight Lines

1. Remove curved-edge default configuration from `avmatrix-web/src/hooks/useSigma.ts`.
2. Remove edge-specific `type: 'curved'` and `curvature` stamping from `avmatrix-web/src/lib/graph-adapter.ts`.
3. Ensure Sigma falls back to standard straight-line edge rendering with no semantic regressions.

Expected outcome:

- the graph keeps the same relationships and filters
- only the rendered geometry changes from curved to straight

### Workstream B — Reduce Selected-Context Edge Thickness To 70% In Focused Mode

1. Leave ambient edge sizing unchanged in `avmatrix-web/src/lib/graph-adapter.ts`.
2. Reduce only the selected-context edge thickness used when `all graph links` are off and a node is selected.
3. Keep the full-graph ambient mode visually unchanged.
4. Do not apply this reduction to AI/query/blast-radius highlight edges in the first pass.
5. Implement this as a conditional render-width rule, not as a permanent change to base edge size data.

Expected outcome:

- the focused single-node mode feels lighter and less tiring
- the full-graph mode keeps its current readability and information density

### Workstream C — Validate Edge State Contrast

After A and B are applied, verify that these states still read clearly:

1. no selection, ambient links on
2. selected node while ambient links are on
3. selected node while ambient links are off
4. AI/query/blast-radius highlight states
5. graph links off + selected node context visible

If contrast is too weak after thinning the focused selected-context links, adjust only that focused selected-context styling, not the global ambient sizing rule.

## Validation

### Static Validation

Run:

```bash
cd avmatrix-web && npx tsc -b --noEmit
cd avmatrix-web && npm test
```

### Manual Validation

Use the real web UI on at least `AVmatrix-main`.

Verify:

1. Graph links render as straight lines.
2. Full-graph mode keeps its current visual density and thickness.
3. With ambient links on, selecting a node does not make its contextual links thinner than today.
4. With `all graph links` off, selecting a node still makes its direct links readable immediately, and those links feel lighter than before.
5. Turning graph links off still hides ambient links, while selected-node contextual links still behave according to the current semantics.
6. Highlight-driven states remain distinguishable from selected-context links.

## Risks

1. Straight lines can increase visible crossings in some areas, even while reducing the “curved spaghetti” effect.
2. If the focused selected-context links are thinned too aggressively, some relationship types may become too faint in that mode.
3. Selected-context and highlight states may need a small secondary tuning pass to preserve visual hierarchy.
4. If the conditional width logic is patched directly into `useSigma.ts` without discipline, edge-rendering logic may become harder to maintain; if that starts happening, the width policy should be extracted into a helper in parallel.

## Rollout Order

1. Switch curved edges to straight lines.
2. Reduce selected-context link thickness to 60% in the focused mode.
3. Validate selection/highlight readability across both ambient-on and ambient-off modes.
4. Only if needed, do a small second-pass tuning of selected-context emphasis.

## Success Criteria

1. Users perceive the graph as less visually tiring.
2. Full-graph mode remains unchanged in density and readability.
3. Selected-node contextual mode feels lighter without losing relationship readability.
4. Selected-node and highlight states remain clearly legible.
5. No behavior regressions in graph filtering, selection, or link-visibility toggles.
