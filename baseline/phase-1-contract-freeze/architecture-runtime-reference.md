# Phase 1 Architecture Runtime Reference

Date: 2026-05-08

Active implementation repository: `F:/AVmatrix-GO`
Architecture/runtime-flow reference: `F:/avmatrix-main`
Active baseline commit: `f2b7e6a5a966c100387566fa8d73208f233ac6bd`
Reference commit read: `79103232acb738d1536bc58e0c7ca7bb261d783d`

Decision rules:

- The active repository remains the implementation workspace.
- The reference repository is used only to compare architecture shape and runtime flow.
- If source-level surfaces differ, classify the mismatch before writing Go code.

Reference comparison:

- cliCommands: no mismatch detected
- httpRoutes: no mismatch detected
- mcpTools: no mismatch detected
- pipelinePhases: no mismatch detected

Frozen surfaces in this batch:

- CLI command and flag shape
- HTTP route method/path shape
- MCP tool/resource/template/prompt names
- Analyzer phase order
- Environment variable names
- Graph schema authority candidates
- Supported language matrix and fixture counts
- Analyze metrics and repo registry/meta shape
