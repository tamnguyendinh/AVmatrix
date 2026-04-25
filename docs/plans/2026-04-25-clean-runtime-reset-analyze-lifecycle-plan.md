# Clean Runtime Reset And Analyze Lifecycle Plan

Date: 2026-04-25  
Status: Draft  
Scope: launcher reset, backend runtime cleanup, analyze/re-analyze clean rebuild lifecycle

## Goal

Fix the root lifecycle problem behind repo switch hangs, stale graph reads, and LadybugDB lock/WAL errors.

Core rule:

```text
any flow that reaches analyze must clean old runtime/index/process state thoroughly before analyze starts.

reset/analyze/re-analyze/start selection/dropdown repo switch
-> clean thoroughly
-> analyze from source
-> load graph only from fresh analyze output
```

The five urgent problems:

```text
1. Reset must be thorough.
2. Analyze must cancel/close all active runtime state for the target repo before rebuilding.
3. Re-analyze must use the same clean lifecycle as analyze.
4. Start + select repo must clean old generated data before analyze.
5. Dropdown repo switch must clean old generated data before analyze and must not be blocked by background embedding using the analyze/DB lock incorrectly.
```

Shared principle:

```text
clean thoroughly, do not add marker/flag/cache state to explain old dirty state
```

## Non-Goals

- Do not change graph semantics or graph output.
- Do not change parser/analyze output semantics.
- Do not add snapshot/cache behavior.
- Do not add marker files or extra "requiresAnalyze" state to paper over old indexes.
- Do not add a global one-repo limitation for normal reading.
- Do not redesign graph links or graph rendering.
- Do not silently delete user source code.

## Problem 1: Reset Must Be Thorough

Current bad shape:

```text
Reset
-> may stop some processes
-> stale backend/DB/session state can remain
-> Start opens UI again
-> selecting repo can read old/dirty DB state
-> graph load can fail or hang
```

Required behavior:

```text
Reset
-> stop launcher-owned backend/static runtime
-> terminate owned child processes
-> clear launcher runtime state
-> clear backend runtime sessions if reachable
-> remove Web UI readable index/runtime artifacts created by AVmatrix
-> do not auto-start
```

After reset:

```text
Start AVmatrix
-> open UI clean
-> selecting any repo must trigger analyze from the beginning
-> graph loads only after analyze completes
```

Important:

```text
Reset is not Start.
Reset is not a soft refresh.
Reset means clean runtime ground.
```

Reset cleanup rule:

```text
remove old AVmatrix-generated runtime/index artifacts
do not create a new marker saying old artifacts are invalid
```

Reset cleanup boundary:

```text
allowed to remove:
-> AVmatrix-generated loadable graph/index artifacts for a repo
-> LadybugDB files generated for indexed repos
-> generated graph/index metadata used only to serve old Web UI graph state
-> temporary runtime state owned by the launcher/backend

not allowed to remove:
-> source repository files
-> source repository config
-> repo pointer data needed for the UI to let the user select a repo path again
-> global user config unrelated to indexed repo storage
-> launcher binary
-> bundled backend binary
-> bundled web-dist
```

Reset must not delete:

```text
source repository files
user source project config
unrelated Node/Codex/dev processes
global tool installation files
```

Repo pointer vs loadable index:

```text
repo pointer:
-> repo name/path used so the UI can show a selectable repo
-> may remain after reset
-> must not be treated as graph-ready

loadable index:
-> lbug / lbug.wal / lbug.lock
-> generated graph DB/index artifacts
-> generated metadata that makes Web UI believe graph can be loaded
-> must be removed before analyze can create a fresh graph
```

After reset:

```text
repo may still appear as a selectable path
but graph must not load from old index
selecting repo must clean generated index artifacts and analyze from source
```

## Registry Contract

The global registry is a repo pointer list, not proof that a graph is ready.

```text
registry entry:
-> repo name/path for selection
-> may survive reset cleanup
-> must not by itself prove graph readiness

storage meta/lbug:
-> generated index state
-> proof that graph may be loadable only when present and fresh from current analyze
-> removed by reset/clean-before-analyze cleanup
```

After reset cleanup:

```text
registry entry may remain
storage meta/lbug should be absent
/api/repos may still show repo pointers
/api/repo must not use stale registry indexedAt/stats as graph-ready proof
/api/graph must not load from registry pointer alone
```

UI contract:

```text
repo pointer in /api/repos is selectable source path state
repo pointer is not necessarily indexed/graph-ready state
UI must not imply a pointer-only repo is already indexed after cleanup
graph-ready UI state exists only after fresh analyze writes new generated index data
```

Analyze completion contract:

```text
cleanup removes graph-ready meta/lbug before analyze
analyze completion writes fresh meta/lbug
analyze completion updates registry stats/indexedAt from fresh output
only then can /api/repo and /api/graph treat the repo as graph-ready
```

Delete repo vs reset cleanup:

```text
DELETE /api/repo:
-> removes generated index storage
-> unregisters repo pointer
-> repo disappears from UI list

reset cleanup / clean-before-analyze:
-> removes generated loadable index/runtime artifacts
-> preserves repo pointer/path data
-> repo can still be selected
-> selecting repo runs analyze from source
```

Do not reuse full delete-repo semantics for reset cleanup unless the implementation explicitly splits "remove generated index" from "unregister repo pointer".

## Problem 2: Analyze Must Own The Repo DB Exclusively

Analyze is the clean rebuild path.

Operation classes:

```text
graph/query/search:
-> read graph/index state

embedding:
-> optional background DB writer
-> may hold LadybugDB/session state
-> must not be treated as a graph reader

analyze/clean/delete:
-> exclusive lifecycle writer
-> owns cleanup and rebuild of generated index state
```

Required rule:

```text
before analyze starts for repo X
-> stop accepting old readers for repo X
-> abort graph/query/search streams for repo X
-> cancel or release embedding for repo X if it holds DB/session state
-> close LadybugDB handles for repo X
-> remove old repo X AVmatrix-generated index/runtime artifacts
-> rebuild DB/index from scratch
```

Bad shape to eliminate:

```text
repo X graph stream still reading DB
-> analyze starts and rebuilds DB
-> old reader keeps native handle/session alive
-> LadybugDB lock/WAL/session state becomes unreliable
```

Correct shape:

```text
analyze(repo X)
-> clean repo X runtime/index state
-> restart backend 
-> run analyze from the beginning
-> write fresh repo X DB/index
-> graph load can read only fresh DB/index
```

This must not block unrelated repo Y unless shared process/native DB state makes it unavoidable.

## Problem 3: Re-analyze Must Use The Same Lifecycle

Re-analyze cannot be a separate shortcut that bypasses analyze cleanup.

Required behavior:

```text
re-analyze(repo X)
-> same clean lifecycle as analyze(repo X)
-> clean repo X runtime/index state
-> restart backend 
-> run analyze from the beginning
-> write fresh repo X DB/index
-> reload graph from fresh DB
```

Important:

```text
re-analyze must not run while graph/query/search readers still hold repo X DB.
re-analyze must cancel or release embedding if embedding holds repo X DB/session state.
re-analyze must not leave old graph stream alive.
re-analyze must not silently reuse dirty DB handles.
re-analyze must not have a separate shortcut lifecycle.
```

## Problem 4: Start + Select Repo Must Pre-Clean Before Analyze

Start AVmatrix opens the UI. When the user selects a repo, the app must not trust any old generated data that is still present from a previous runtime/session.

Required behavior:

```text
Start AVmatrix
-> user selects repo X
-> use repo pointer only to know source path
-> immediately check whether repo X has old AVmatrix-generated loadable index/runtime data
-> if old generated index data exists, clean it thoroughly
-> run analyze from the beginning
-> load graph only from the fresh analyze result
```

Important:

```text
repo pointer is not graph-ready state
old readable index is not a reason to skip analyze
old readable index is a reason to clean first
```

This is the same root rule as reset/analyze/re-analyze:

```text
do not reuse dirty generated state
clean generated state first, then analyze from source
```

## Problem 5: Dropdown Repo Switch Must Clean Before Analyze And Must Not Be Blocked By Background Embedding

Current bad shape:

```text
repo A graph loads
-> frontend auto-starts background embedding for repo A
-> /api/embed holds the same activeRepoPaths lock used by analyze
-> user switches repo from dropdown
-> /api/graph sees activeRepoPaths and reports "Repository is being analyzed"
-> UI stays at "Downloading graph... 0.0 MB downloaded" or returns to repo selection
```

Observed evidence:

```text
/api/repos reports graphReady=true for Website and Restaurant_manager
/api/graph?repo=Website&stream=true returns 409 "Repository is being analyzed"
server log shows [embed] ... nodes already embedded
Website lbug.wal grows while background embedding is active
```

Root cause:

```text
activeRepoPaths currently mixes different operations:
-> analyze / clean rebuild
-> embedding
-> delete

assertRepoCanReadGraph treats any activeRepoPaths lock as analyze.
That is wrong.
```

Required behavior:

```text
dropdown repo switch
-> abort current tab graph load
-> cancel background embedding for the repo being left and wait until DB/session is released
-> clean selected repo generated index/runtime artifacts
-> run analyze for selected repo from the beginning
-> load selected repo graph only from fresh analyze output
-> stale embedding/graph response cannot overwrite selected repo state
```

Dropdown switch contract:

```text
header dropdown switch is a repo activation flow
repo activation reaches analyze
therefore it must clean thoroughly before analyze

graphReady may be displayed as current state
but graphReady must not be used to skip clean/analyze in start/reset/dropdown activation flows
```

Backend rule:

```text
analyze/clean/delete are exclusive write lifecycle operations
embedding is optional background DB writer
graph read must not be rejected as "being analyzed" just because embedding is active
```

If LadybugDB cannot safely read graph while embedding writes:

```text
repo switch must cancel embedding cleanly
then close/release DB handles
then graph load proceeds
```

Important:

```text
do not solve this by adding a global one-repo limit
do not solve this by adding timeout
do not silently skip graph load
do not silently hide embedding failure
```

## Clean Rebuild Gate

Do not introduce a large new lifecycle architecture. Add a minimal cleanup gate around the existing analyze/re-analyze paths.

Required gate:

```text
cleanAndAnalyze(repo X)
-> abort active readers for repo X
-> cancel or release embedding for repo X if it holds DB/session state
-> close DB handles for repo X
-> remove old AVmatrix-generated repo X index/runtime artifacts
-> run analyze from the beginning
```

Reader behavior during clean rebuild:

```text
new graph/query/search readers for repo X
-> rejected or delayed while cleanAndAnalyze(repo X) is running

new embedding work for repo X
-> rejected or canceled while cleanAndAnalyze(repo X) is running
```

## Frontend Behavior

Repo selection after reset:

```text
user selects repo X
-> if repo X has no readable index after reset/cleanup
-> start analyze(repo X)
-> wait for completion
-> load graph
```

Repo selection after Start:

```text
user selects repo X
-> check for old AVmatrix-generated repo X data
-> if found, clean it
-> start analyze(repo X)
-> wait for completion
-> load graph
```

Repo switch while graph is loading:

```text
switch from repo A to repo B
-> abort current tab's repo A load
-> stop background embedding for repo A if it can hold the repo DB
-> clear repo A UI state immediately
-> clean repo B generated index/runtime artifacts
-> analyze repo B from source
-> load repo B graph only from fresh analyze output
-> stale repo A response cannot overwrite UI state
```

Dropdown repo switch frontend flow:

```text
1. User selects repo B.
2. Abort any in-flight graph load for repo A in the current tab.
3. Clear repo A UI state immediately:
   -> graph
   -> selected node
   -> highlights
   -> query result
   -> code references
   -> chat/agent state if bound to repo A
   -> embedding UI state bound to repo A
4. Set UI to analyzing/loading repo B.
5. Start backend clean-before-analyze for repo B.
6. Ignore any late response from repo A by checking active repo request id.
7. Load repo B graph only after repo B analyze completes.
```

Abort guard remains useful, but it is not the root fix by itself.

## Backend Behavior

Backend must provide one clear cleanup path before analyze/re-analyze writes a new index.

Responsibilities:

```text
track active graph/query/search readers only as needed for cleanup
track embedding as background DB writer, not as reader
abort same-repo readers before clean rebuild starts
cancel/release same-repo embedding before closing LadybugDB handles
close LadybugDB handles before clean rebuild starts
remove old generated index/runtime artifacts before analyze writes new ones
reject or delay same-repo readers while clean rebuild is running
clear stale generated loadable index state after reset
preserve enough repo pointer/path data for the user to select repos again
```

Operation lock responsibilities:

```text
separate lock kinds:
-> analyze/clean/delete exclusive lock
-> embedding background write lock
-> graph/query/search read tracking

graph/query/search must know which lock blocks it and why.
Do not report embedding as analyze.
```

Graph/query/search endpoints:

```text
keep existing behavior
-> add only the minimal abort/cleanup hook needed by clean rebuild
```

Embedding endpoint:

```text
must be cancelable by repo switch and clean rebuild
must release DB/session lock on cancel/error/complete
must not keep stale activeRepoPaths state after client leaves
must not block unrelated repo graph loads unless LadybugDB native state forces that and it is documented
```

Analyze/re-analyze endpoints:

```text
clean repo generated state
-> run analyze job
-> expose graph only after analyze completes
```

Repo selection/start-analyze path:

```text
selected repo has old generated index/runtime artifacts
-> clean repo generated state
-> run analyze job
-> expose graph only after analyze completes
```

Dropdown repo switch backend flow:

```text
1. Receive activation request for repo B.
2. Stop accepting new work for repo B during clean/analyze.
3. Cancel or release graph/query/search readers for repo B.
4. Cancel or release embedding job for repo B if it holds DB/session state.
5. Close repo B LadybugDB handles.
6. Remove repo B generated index/runtime artifacts.
7. Run analyze repo B from source.
8. Reinitialize backend repo registry/session state.
9. Serve repo B graph only from fresh analyze output.
```

Reset endpoint or launcher reset integration:

```text
abort all readers
cancel/stop analyze jobs if needed
cancel/stop embedding jobs if needed
close all DB handles
clear runtime session state
remove AVmatrix-generated loadable runtime/index artifacts readable by Web UI
preserve repo pointer/path data needed for selection
```

Reset order:

```text
1. Stop accepting new graph/query/search/embed/analyze work.
2. Abort graph/query/search readers.
3. Cancel analyze jobs.
4. Cancel embedding jobs.
5. Wait for DB/session release.
6. Close DB handles.
7. Remove generated index/runtime artifacts.
8. Preserve repo pointers.
```

## Validation

Manual validation:

```text
1. Start AVmatrix.
2. Load repo A.
3. Reset.
4. Start AVmatrix again.
5. Select repo A.
6. Expected: repo A analyzes from the beginning, then graph loads.
7. Select repo B.
8. Expected: repo B analyzes from the beginning because reset removed old readable indexes, then graph loads.
```

Start selection validation:

```text
1. Start AVmatrix.
2. Select repo A that already has old generated index/runtime artifacts.
3. Expected: old generated artifacts are cleaned first.
4. Expected: repo A analyzes from the beginning.
5. Expected: graph loads only after fresh analyze completes.
```

Analyze validation:

```text
1. Start loading graph for a large repo.
2. Start analyze for the same repo.
3. Expected: old graph/query/search readers are canceled or denied.
4. Expected: background embedding is canceled or releases the DB/session lock.
5. Expected: old generated index/runtime artifacts are removed before rebuild.
6. Expected: analyze rebuilds cleanly.
7. Expected: graph loads after analyze completes.
```

Re-analyze validation:

```text
1. Load repo graph.
2. Start re-analyze from UI.
3. Expected: current readers for that repo are canceled/closed.
4. Expected: old generated index/runtime artifacts are removed before rebuild.
5. Expected: re-analyze runs cleanly.
6. Expected: UI reloads graph from the new DB only.
```

Repo switch with embedding validation:

```text
1. Load repo A.
2. Let auto embedding start.
3. Switch to repo B from the header dropdown.
4. Expected: repo A embedding is canceled or releases the DB before repo B analyze starts.
5. Expected: repo B old generated index/runtime artifacts are cleaned before analyze.
6. Expected: repo B analyzes from source.
7. Expected: /api/graph for repo B loads only after fresh analyze output exists.
8. Expected: /api/graph for repo B does not return "Repository is being analyzed" because repo A embedding was active.
9. Expected: UI leaves "Downloading graph... 0.0 MB downloaded" and enters repo B after fresh analyze.
10. Switch back to repo A.
11. Expected: repo A follows the same clean-before-analyze activation flow, with no stale repo overwrite.
```

Negative validation:

```text
no stale repo graph after reset
no graph load from dirty DB after reset
repo pointers may remain after reset, but they are not graph-ready state
no old graph stream overwrites new selected repo
late repo A response cannot overwrite repo B UI state after dropdown switch
no LadybugDB reader remains active while clean rebuild runs for the same repo
no background embedding blocks repo switch as if analyze were running
no graphReady shortcut skips clean-before-analyze in start/reset/dropdown activation flows
no marker/flag/cache workaround required to explain reset state
```

## Implementation Order

1. Add lifecycle instrumentation only:
   ```text
   repo reader start/end/abort
   repo clean-analyze start/end/fail
   DB handle close/open
   reset clear start/end
   ```

2. Add a minimal backend clean rebuild gate around existing analyze/re-analyze:
   ```text
   same-repo reader abort
   same-repo embedding cancel/release
   DB close hook
   generated index/runtime artifact cleanup
   no large lifecycle rewrite
   ```

3. Add minimal reader abort hooks to graph/query/search where they can hold repo DB state.

4. Add minimal embedding cancel/release hook where embedding can hold repo DB/session state.

5. Route analyze/re-analyze through the same clean-analyze lifecycle.

6. Route Start + repo selection through the same clean-before-analyze path when old generated data exists.

7. Update reset to stop readers, analyze jobs, and embedding jobs; clear runtime state thoroughly; remove AVmatrix-generated readable indexes before Web UI graph load; preserve repo pointer/path data for selection.

8. Route dropdown repo switch through clean-before-analyze activation flow.

9. Keep frontend repo switch abort/stale guard as tab-level protection.

10. Split operation locks so background embedding cannot masquerade as analyze:
   ```text
   analyze/clean/delete lock
   embedding lock
   reader tracking
   cancel embedding on same-repo clean rebuild and repo switch when needed
   ```

11. Validate with:
   ```text
   Website
   Restaurant_manager
   AVmatrix-main
   hotel_manager
   ```

## Safety Rules

- Normal multi-repo reading must remain possible.
- Clean-analyze exclusivity applies to the same repo DB first.
- If LadybugDB native limitations force broader locking, document the exact reason before implementing it.
- Never silently hide analyze failures.
- Never silently serve old graph after reset.
- Never add marker/flag/cache state where deletion/cleanup of generated artifacts is the cleaner fix.
- Never delete source repo content.
