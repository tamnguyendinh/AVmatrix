# Phase 1 MCP Contract

Observed: 2026-05-08T21:30:00+07:00

Runtime dump source: `AVMATRIX_TOOLS`, `getResourceDefinitions`, and `getResourceTemplates` from
the built `avmatrix/dist` output. The temporary full dump is `.tmp/phase1_mcp_runtime_dump.json`;
the durable contract is `mcp-contract.json`.

## Tools

Current MCP tools:

- `list_repos`
- `query`
- `cypher`
- `context`
- `detect_changes`
- `rename`
- `impact`
- `route_map`
- `tool_map`
- `shape_check`
- `api_impact`
- `group_list`
- `group_sync`
- `group_contracts`
- `group_query`
- `group_status`

Legacy aliases still route through backend dispatch: `search -> query`, `explore -> context`,
`overview -> overview`.

## Resources

Static resources:

- `avmatrix://repos`
- `avmatrix://setup`

Templates:

- `avmatrix://repo/{name}/context`
- `avmatrix://repo/{name}/clusters`
- `avmatrix://repo/{name}/processes`
- `avmatrix://repo/{name}/schema`
- `avmatrix://repo/{name}/cluster/{clusterName}`
- `avmatrix://repo/{name}/process/{processName}`

## Prompts

- `detect_impact(scope?, base_ref?)`
- `generate_map(repo?)`

## Staleness And Error Behavior

- No indexed repositories: `No indexed repositories. Run: avmatrix analyze`.
- Stale legacy Kuzu index: `AVmatrix: "<repo>" has a stale KuzuDB index. Run: avmatrix analyze <path>`.
- Repo context resource includes `staleness: "<hint>"` when git staleness is detected.
- Repo context resource always includes `re_index: Run avmatrix analyze in terminal if data is stale`.
- FTS degraded warning tells the user to run `avmatrix analyze --force`.

## Response Envelope

MCP tool success returns text content with an appended next-step hint. Tool failure returns text
content beginning with `Error:` and `isError: true`. Resource read failures return text/plain
`Error: <message>`.

The Go MCP server must preserve tool names, schemas, resource URI strings, prompt names, stale-index
warnings, and next-step hint behavior unless a later explicit contract migration is recorded.

