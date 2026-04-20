---
title: "feat: Complete COBOL language feature coverage for maximum knowledge graph value"
type: feat
status: active
date: 2026-03-26
origin: Feature audit from v3-integration-architect agent (session 8642401e)
---

## Enhancement Summary

**Deepened on:** 2026-03-26
**Research agents used:** COBOL expert (Phase 1+2), graph value analyst, codebase explorer
**Sections enhanced:** Phase 1 (5 features), Phase 2 (4 features), graph value ranking

### Key Improvements from Research
1. **CALL USING** is the #1 highest-value edge type (9.2/10) — fixes ~40% of missing caller references
2. **EXEC DLI** requires dual-interface support (EXEC DLI + CBLTDLI CALL) for full IMS coverage
3. **DECLARATIVES** is lowest-risk Phase 2 item — existing section/paragraph detection already captures structure
4. **SET TO TRUE** accounts for 80-90% of all SET statements — prioritize this form
5. **INSPECT** needs multi-line accumulator (like SORT) — can span 5+ continuation lines
6. **Graph value ranking**: cobol-call-using (9.2) > cobol-error-handler (9.0) > dli-gu (8.2) > cobol-string (6.2)

### New Edge Cases Discovered
- CALL USING supports mixed modes: `USING BY REFERENCE WS-A BY CONTENT WS-B BY VALUE WS-C`
- CALL USING `ADDRESS OF` and `OMITTED` must be filtered from parameter lists
- EXEC DLI can have multiple SEGMENT levels in hierarchical retrieval (use matchAll)
- DECLARATIVES can have multiple USE sections (one per file + catch-all for INPUT/OUTPUT/I-O/EXTEND)
- INSPECT TALLYING can have multiple counters in a single statement
- STRING/UNSTRING can span multiple lines (need accumulator pattern)

---

# Complete COBOL Language Feature Coverage

## Overview

Implement the remaining 25 unhandled COBOL language features and fix 10 partial features to achieve ~95% coverage (up from 71.9%). The goal is to build the richest possible knowledge graph from COBOL codebases, enabling a future `modernize` MCP command (out of scope for this plan) that would use the graph to assist with COBOL-to-modern-language migration.

## Problem Statement

The COBOL processor currently handles 54 of 89 applicable language features (71.9%). The 25 unhandled features represent real data loss in the knowledge graph:
- **Cross-program data flow** is invisible (CALL ... USING parameters not extracted)
- **IMS/DB programs** produce empty graphs (EXEC DLI not recognized)
- **String transformation logic** is invisible (STRING/UNSTRING/INSPECT not tracked)
- **SQL copybook dependencies** are missing (EXEC SQL INCLUDE not mapped)
- **Error handling flows** are lost (DECLARATIVES/USE AFTER not captured)

## Proposed Solution

Implement features in 4 phases, ordered by graph value density (edges created per LOC of implementation). Each phase is independently shippable and testable.

## Technical Approach

### Phase 1: High-Value Data Flow Edges (~150 LOC, ~8 new edge types)

The highest-ROI features: they create new ACCESSES and IMPORTS edges that directly improve impact analysis.

**Critical research finding**: Multi-line statement accumulation is the dominant challenge. CALL USING, STRING/UNSTRING, and multi-line data item clauses all span multiple lines in production COBOL. The free-format path processes each line independently — these features need statement accumulators (like SORT/SELECT) or the free-format path needs multi-line awareness. Estimated LOC increased from 110 to 150 to account for accumulator infrastructure.

#### 1.1 EXEC SQL INCLUDE -> IMPORTS edges
- **File:** `cobol-preprocessor.ts` (parseExecSqlBlock)
- **What:** Detect `INCLUDE` as the operation, extract member name, emit as a `copies[]` entry
- **Graph:** IMPORTS edge from File to included copybook/SQLCA with reason `sql-include`
- **Tests:** Unit test for `EXEC SQL INCLUDE SQLCA END-EXEC` and `EXEC SQL INCLUDE CUSTCOPY END-EXEC`

**Research insights (EXEC SQL INCLUDE):**
- DB2 member names can contain underscores: `EXEC SQL INCLUDE CUST_TBL_DCL END-EXEC` — regex must use `[A-Z][A-Z0-9_-]+`
- Quoted literal form: `EXEC SQL INCLUDE 'DBRMLIB.MEMBER' END-EXEC` (z/OS PDS qualified name)
- SQLCA/SQLDA are DB2 builtins — won't resolve to repo files. Emit unresolved IMPORTS edge (still valuable)
- No REPLACING support on EXEC SQL INCLUDE (unlike COPY)
- Add `INCLUDE` to `OP_MAP` in `parseExecSqlBlock`; extract member via `RE_SQL_INCLUDE = /^INCLUDE\s+(?:'([^']+)'|"([^"]+)"|([A-Z][A-Z0-9_-]+))/i`

#### 1.2 CALL ... USING parameter extraction -> ACCESSES edges (Graph value: 9.2/10)
- **File:** `cobol-preprocessor.ts` (processLogicalLine CALL section)
- **What:** After capturing CALL target, scan for USING clause. Extract parameter names (reuse USING_KEYWORDS filter). Store as `calls[].parameters: string[]`
- **Interface:** Add `parameters?: string[]` to calls array type in CobolRegexResults
- **File:** `cobol-processor.ts` (CALL edge block)
- **Graph:** For each USING parameter, create ACCESSES edge from caller to data item Property node with reason `cobol-call-using`
- **Tests:** `CALL 'AUDITLOG' USING CUST-ID WS-AMOUNT` -> 2 ACCESSES edges

**Research insights (CALL USING forms):**
- Mixed modes: `CALL 'PGM' USING BY REFERENCE WS-A BY CONTENT WS-B BY VALUE WS-C`
- Pointer passing: `CALL 'PGM' USING ADDRESS OF WS-A`
- Placeholder: `CALL 'PGM' USING OMITTED WS-B`
- Filter keywords: add `ADDRESS`, `OMITTED`, `LENGTH` to USING_KEYWORDS (already has BY/VALUE/REFERENCE/CONTENT)
- **Impact tool enhancement:** CALL-USING edges enable BFS traversal through parameter data flow — single most impactful edge type for COBOL impact analysis

#### 1.3 STRING/UNSTRING data flow -> ACCESSES edges
- **File:** `cobol-preprocessor.ts` (new section in extractProcedure)
- **What:** Accumulate multi-line STRING/UNSTRING until period or END-STRING/END-UNSTRING. Extract sources and INTO targets.
- **Interface:** Add `strings: Array<{ sources: string[]; target: string; type: 'string' | 'unstring'; line: number; caller: string | null }>` to CobolRegexResults
- **Graph:** read-ACCESSES on sources, write-ACCESSES on INTO target with reason `cobol-string-read` / `cobol-string-write`
- **Tests:** 2 unit tests + integration test assertions

**Research insights (STRING/UNSTRING):**
- **Needs statement accumulator** — STRING/UNSTRING always span multiple lines in production
- Terminate accumulation at: period, END-STRING/END-UNSTRING, or start of next COBOL verb
- STRING sources: identifiers before each `DELIMITED BY`. Filter: STRING, DELIMITED, BY, SIZE, ALL, INTO, WITH, POINTER, ON, OVERFLOW, NOT, END-STRING
- UNSTRING: source is first identifier after UNSTRING; INTO targets are identifiers after INTO. Filter: DELIMITER, IN, COUNT, TALLYING, OR
- WITH POINTER field is both read AND written (starting position updated)
- TALLYING IN / COUNT IN fields are write targets
- Literal sources (`'text'`) must be filtered — quote-aware tokenization needed
- **Edge case**: STRING terminated by next verb, not period — existing fixture has `STRING ... DISPLAY` without period between them

#### 1.4 OCCURS DEPENDING ON -> ACCESSES edge
- **File:** `cobol-preprocessor.ts` (parseDataItemClauses)
- **What:** Extend OCCURS regex to capture DEPENDING ON field, KEY fields, and INDEXED BY names
- **Interface:** Add `dependingOn?: string`, `occursMax?: number`, `occursKeys?: Array<{direction: string; fields: string[]}>`, `indexedBy?: string[]` to data items
- **Graph:** ACCESSES edge from table item to controlling field with reason `cobol-depends-on`
- **Tests:** `05 WS-TABLE OCCURS 100 DEPENDING ON WS-COUNT` -> edge

**Research insights (OCCURS):**
- IBM allows `OCCURS 0 TO n DEPENDING ON` (zero minimum) and `OCCURS UNBOUNDED DEPENDING ON` (V6.4)
- Subscripted controlling fields: `DEPENDING ON WS-COUNT(WS-IDX)` — strip subscripts before storing
- **Pre-existing gap**: Multi-line data item clauses without continuation indicator are NOT captured. `05 WS-TABLE\n    OCCURS 100\n    DEPENDING ON WS-COUNT.` — the current RE_DATA_ITEM only gets the first line, `rest` is empty. Fixing properly requires a data item accumulator (like SELECT). **Defer full fix to Phase 3; implement same-line capture now.**
- KEY IS fields: `ASCENDING KEY IS WS-KEY-1 WS-KEY-2` — capture for SEARCH ALL resolution
- INDEXED BY: `INDEXED BY IDX-1 IDX-2` — capture for SET/SEARCH context

#### 1.5 VALUE clause for standard data items
- **File:** `cobol-preprocessor.ts` (parseDataItemClauses)
- **What:** Extract VALUE using a pragmatic function that handles quoted strings, numerics, figurative constants, hex/national literals
- **Interface:** Already exists as `values?: string[]` on data items (currently only populated for 88-level)
- **Graph:** Stored in Property node description (no new edges)
- **Tests:** `01 WS-STATUS PIC X VALUE 'A'` -> values: ['A']

**Research insights (VALUE forms):**
- Hex literals: `VALUE X'F1F2F3F4'`, National: `VALUE N'text'`, DBCS: `VALUE G'text'`
- Figurative constants: SPACES, ZEROS, ZEROES, LOW-VALUES, HIGH-VALUES, QUOTES, NULL, NULLS
- ALL literal: `VALUE ALL '*'`
- Numeric with sign/decimal: `VALUE -123.45`, `VALUE +1`
- `VALUE IS` optional — both `VALUE 'A'` and `VALUE IS 'A'` valid
- **Decimal vs period ambiguity**: `VALUE 100.` — is `.` decimal or terminator? `parseDataItemClauses` already strips trailing period, so this is handled
- IBM V6.4: floating-point `VALUE 1.0E5` — extend numeric regex if needed
- Implementation: use a pragmatic `extractValue(rest)` function, not a single complex regex

### Phase 2: EXEC DLI + DECLARATIVES (~90 LOC, ~4 new edge types)

IMS/DB support and error handling flows.

#### 2.1 EXEC DLI (IMS/DB) -> ACCESSES edges (Graph value: 8.2/10)
- **File:** `cobol-preprocessor.ts` (processLogicalLine — add RE_EXEC_DLI_START check alongside SQL/CICS)
- **What:** Accumulate EXEC DLI blocks like EXEC SQL. Parse DLI verbs (GU, GN, GNP, GHU, GHN, GHNP, ISRT, DLET, REPL, CHKP, SCHD, TERM). Extract segment name, PCB number, INTO/FROM areas, WHERE fields, PSB name.
- **Interface:** Add `execDliBlocks: Array<{ line: number; verb: string; pcbNumber?: number; segmentName?: string; intoField?: string; fromField?: string; whereField?: string; psbName?: string }>` to CobolRegexResults
- **Graph:** CodeElement node + ACCESSES edge to `<ims>:<segmentName>` Record node with reason `dli-{verb}`; ACCESSES edges to INTO/FROM data areas; PSB ACCESSES for SCHD
- **Tests:** `EXEC DLI GU USING PCB(1) SEGMENT(CUSTOMER) INTO(WS-CUST) END-EXEC`

**Research insights (dual IMS interface):**
- **EXEC DLI**: Embedded command interface for CICS-DL/I programs only
- **CBLTDLI CALL**: Batch interface via `CALL 'CBLTDLI' USING function-code PCB io-area SSA1..SSA15`
- CBLTDLI is already captured as a CALL to 'CBLTDLI' — enrich with USING parameter semantics later
- Multiple SEGMENT levels in hierarchical retrieval — use `matchAll` on segment regex
- DLI verbs: GU (most common), GN, GNP, GHU, GHN, GHNP, ISRT, REPL, DLET, CHKP, SCHD, TERM, ROLL, ROLB
- **Edge case**: DLET/REPL have no SEGMENT clause (operate on current position)
- **Recommended order**: Implement AFTER DECLARATIVES and SET (lower risk, higher frequency)

#### 2.2 DECLARATIVES / USE AFTER STANDARD EXCEPTION (Graph value: 9.0/10)
- **File:** `cobol-preprocessor.ts` (processLogicalLine — detect DECLARATIVES keyword, track USE AFTER blocks)
- **What:** When `DECLARATIVES.` is encountered, switch to declaratives mode. Extract USE statements binding sections to files/modes.
- **Interface:** Add `declaratives: Array<{ sectionName: string; useType: 'error' | 'debug' | 'label' | 'reporting'; target: string; line: number }>` to CobolRegexResults
- **Graph:** ACCESSES edge from declarative Namespace to file Record with reason `cobol-declarative-error-handler`
- **Tests:** Unit test with DECLARATIVES section, integration test for error flow

**Research insights (DECLARATIVES syntax):**
- `USE AFTER STANDARD {EXCEPTION|ERROR} ON {file-name|INPUT|OUTPUT|I-O|EXTEND}`
- EXCEPTION and ERROR are synonymous; STANDARD is optional in IBM dialects
- Multiple USE sections allowed (one per file + catch-all for I/O modes)
- `END DECLARATIVES.` must NOT reset PROCEDURE DIVISION state
- `DECLARATIVES` is already in EXCLUDED_PARA_NAMES — no false paragraph risk
- Existing section/paragraph detection already captures structural elements — just need USE binding
- **Lowest risk Phase 2 item** — implement first

#### 2.3 SET statement -> ACCESSES edges
- **File:** `cobol-preprocessor.ts` (extractProcedure — new RE_SET regex)
- **Interface:** Add `sets: Array<{ targets: string[]; form: 'to-true'|'to-value'|'up-by'|'down-by'|'address-of'|'to-null'|'to-entry'; value?: string; entryTarget?: string; entryIsLiteral?: boolean; line: number; caller: string | null }>` to CobolRegexResults
- **Graph:** ACCESSES write edge with reason `cobol-set-condition` (TO TRUE), `cobol-set-index` (TO/UP/DOWN), `cobol-set-address` (ADDRESS OF). SET ENTRY with literal -> CALLS edge.
- **Tests:** `SET WS-EOF TO TRUE`, `SET IDX-1 TO 5`, `SET IDX-1 UP BY 1`

**Research insights (SET forms by frequency):**
- `SET condition TO TRUE` — 80-90% of all SET usage. Multiple targets: `SET COND-A COND-B TO TRUE`
- `SET index TO/UP BY/DOWN BY` — ~8%. Multiple indices: `SET IDX-1 IDX-2 UP BY 1`
- `SET pointer TO ADDRESS OF data-item` / `SET ADDRESS OF data-item TO pointer` — ~2%
- `SET proc-ptr TO ENTRY "PROGNAME"` — rare but creates CALLS edge (like dynamic CALL)
- Filter OF/IN qualifiers: `SET COND-A OF WS-RECORD TO TRUE` (strip OF WS-RECORD)
- **Prioritize**: SET TO TRUE alone covers 80-90% — implement this form first

#### 2.4 INSPECT -> ACCESSES edges
- **File:** `cobol-preprocessor.ts` (extractProcedure — new `inspectAccum` accumulator like SORT)
- **What:** Accumulate multi-line INSPECT until period. Extract inspected field + tally counters.
- **Interface:** Add `inspects: Array<{ inspectedField: string; counters: string[]; form: 'tallying'|'replacing'|'converting'|'tallying-replacing'; line: number; caller: string | null }>` to CobolRegexResults
- **Graph:** ACCESSES read on inspected field always; write if REPLACING/CONVERTING. Write edges for tally counters. Reason: `cobol-inspect-read`/`cobol-inspect-write`/`cobol-inspect-tally`
- **Tests:** `INSPECT WS-FIELD TALLYING WS-COUNT FOR ALL 'A'` -> read on WS-FIELD, write on WS-COUNT

**Research insights (INSPECT forms by frequency):**
- REPLACING (~60%): `INSPECT WS-STR REPLACING ALL 'A' BY 'B'`
- TALLYING (~25%): `INSPECT WS-STR TALLYING WS-CNT FOR ALL 'A'` — multiple counters possible
- CONVERTING (~10%): `INSPECT WS-STR CONVERTING 'abc' TO 'ABC'`
- Combined (~5%): TALLYING + REPLACING in single statement
- **Needs multi-line accumulator** — INSPECT frequently spans 3-5 lines in production
- Extract tally counters with `([A-Z][A-Z0-9-]+)\s+FOR\b` matchAll pattern
- Filter figurative constants (SPACES, ZEROS) using existing MOVE_SKIP set

### Phase 3: Completeness Fixes (~60 LOC)

Fix the 10 partial features and small gaps.

#### 3.1 CALL ... RETURNING extraction
- Extend RE_CALL processing to capture RETURNING target after the USING clause
- Store as `calls[].returning?: string`
- Graph: ACCESSES write edge with reason `cobol-call-returning`

#### 3.2 SELECT OPTIONAL flag preservation
- Store `isOptional: boolean` in FileDeclaration interface
- Include in Record node description

#### 3.3 ALTERNATE RECORD KEY extraction
- Add regex in parseSelectStatement: `/\bALTERNATE\s+RECORD\s+KEY\s+(?:IS\s+)?([A-Z][A-Z0-9-]+)/i`
- Store as `alternateKeys?: string[]`

#### 3.4 COMMON attribute on nested programs
- Extend RE_PROGRAM_ID: `/\bPROGRAM-ID\.\s*([A-Z][A-Z0-9-]+)(?:\s+IS\s+COMMON)?/i`
- Store `isCommon: boolean` on Module node
- Affects cross-program CALL resolution scope

#### 3.5 IS EXTERNAL / IS GLOBAL as first-class properties
- Change from usage string hack to proper boolean fields on data items
- Add `isExternal?: boolean`, `isGlobal?: boolean` to data item interface

#### 3.6 AUTHOR / DATE-WRITTEN mapped to Module node
- Already extracted as programMetadata — map to Module node properties
- `graph.addNode({ ..., properties: { ..., author, dateWritten } })`

#### 3.7 REPLACE statement
- Track REPLACE / REPLACE OFF state in preprocessor
- Apply text substitutions during preprocessing (before regex extraction)
- Complex: requires careful scoping rules

### Phase 4: Niche Features (~30 LOC)

Low-priority but nice for completeness.

#### 4.1 INITIALIZE statement -> write ACCESSES
- `/\bINITIALIZE\s+([A-Z][A-Z0-9-]+)/i`
- ACCESSES write edge with reason `cobol-initialize`

#### 4.2 Remaining IDENTIFICATION DIVISION paragraphs
- DATE-COMPILED, INSTALLATION, SECURITY, REMARKS
- Map to Module node description properties

#### 4.3 EXEC SQL INCLUDE -> IMPORTS edge (expansion)
- For EXEC SQL INCLUDE inside EXEC blocks that reference copybooks containing SQL
- Create IMPORTS edge similar to COPY

## Acceptance Criteria

### Functional Requirements

- [ ] Phase 1: All 5 features implemented with unit + integration tests
- [ ] Phase 2: All 4 features implemented with unit + integration tests
- [ ] Phase 3: All 7 partial features fixed
- [ ] Phase 4: At least 2 of 3 niche features implemented
- [ ] All existing 145 tests continue to pass
- [ ] TypeScript compiles cleanly

### Non-Functional Requirements

- [ ] No performance regression: CardDemo benchmark stays under 8s
- [ ] No file exceeds 1500 LOC (preprocessor currently 1326)
- [ ] ACAS benchmark shows increased node/edge counts (more data extracted)
- [ ] CardDemo benchmark shows increased edge counts (CALL USING, STRING, etc.)

### Quality Gates

- [ ] Each phase has its own commit
- [ ] Integration test assertions updated with exact counts per phase
- [ ] Benchmark run after each phase to track graph growth

## Dependencies & Risks

### Dependencies
- None. All changes are additive to existing COBOL processor code.
- No LanguageProvider changes needed.
- No graph schema changes needed (all new constructs map to existing node labels + edge types).

### Risks
- **preprocessor.ts size**: Currently 1326 LOC. Phase 1+2 adds ~200 LOC -> 1526 LOC. May need to extract helpers into a separate `cobol-data-flow.ts` module if it exceeds 1500.
- **REPLACE statement** (Phase 3.7) is the most complex feature — requires tracking text substitution state across logical lines. Consider deferring to a separate PR if it takes >100 LOC.
- **EXEC DLI** (Phase 2.1) is only testable against IMS codebases. Need fixture data or synthetic test cases.

## Graph Value Ranking by MCP Tool Impact

Research agent analyzed all 5 MCP tools (query, context, impact, detect_changes, rename) against planned edge types:

| Edge Type | QUERY | CONTEXT | IMPACT | DETECT | RENAME | **Overall** |
|-----------|-------|---------|--------|--------|--------|-------------|
| `cobol-call-using` | 4/5 | 5/5 | 5/5 | 4/5 | 4/5 | **9.2/10** |
| `cobol-error-handler` | 5/5 | 4/5 | 5/5 | 5/5 | 2/5 | **9.0/10** |
| `dli-*` (IMS verbs) | 4/5 | 4/5 | 5/5 | 4/5 | 2/5 | **8.2/10** |
| `cobol-string-*` | 4/5 | 3/5 | 3/5 | 3/5 | 2/5 | **6.2/10** |

**Key finding**: `cobol-call-using` alone would fix ~40% of missing caller references in COBOL graphs.

## Future Considerations

This plan provides the graph data foundation for a future `modernize` MCP command (out of scope) that would:
- Use CALL USING edges to map data contracts between programs
- Use STRING/UNSTRING edges to identify data transformation logic
- Use EXEC SQL/DLI edges to map database access patterns
- Use DECLARATIVES to understand error handling architecture
- Use the complete knowledge graph to generate migration plans

**MCP tool enhancements needed** (after this plan ships):
- Add `cobol-call-using`, `cobol-error-handler`, `dli-*` to IMPACT tool's default `relationTypes` for COBOL repos
- Add confidence floors for new edge types in `IMPACT_RELATION_CONFIDENCE`
- Register new edge types in `VALID_RELATION_TYPES` set (`local-backend.ts:52`)

## Sources & References

### Internal References
- Feature audit: session 8642401e (COBOL expert agent, 123 features audited)
- Prior plans: `docs/plans/2026-03-25-feat-cobol-100-percent-feature-coverage-plan.md`
- Architecture: `docs/code-indexing/cobol/` (7 documentation files)

### External References
- COBOL features reference: mainframestechhelp.com/tutorials/cobol/features.htm
- COBOL-85 standard: ISO/IEC 1989:1985
- IBM Enterprise COBOL reference
