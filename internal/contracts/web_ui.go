package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugschema"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

const (
	WebUIContractSchemaPath     = "contracts/web-ui/avmatrix-web-contract.schema.json"
	WebUIContractTypeScriptPath = "avmatrix-web/src/generated/avmatrix-contracts.ts"
)

type LanguageContract struct {
	Name       string   `json:"name"`
	Value      string   `json:"value"`
	Extensions []string `json:"extensions"`
	Syntax     string   `json:"syntax"`
}

type WebUIContractManifest struct {
	Status        string `json:"status"`
	GeneratedFrom string `json:"generatedFrom"`
	Artifacts     struct {
		SchemaManifest    string `json:"schemaManifest"`
		TypeScriptAdapter string `json:"typescriptAdapter"`
	} `json:"artifacts"`
	Graph struct {
		NodeLabels                 []string `json:"nodeLabels"`
		GraphRelationshipTypes     []string `json:"graphRelationshipTypes"`
		LadybugDBNodeTables        []string `json:"ladybugdbNodeTables"`
		LadybugDBRelationshipTypes []string `json:"ladybugdbRelationshipTypes"`
		RelationshipTableName      string   `json:"relationshipTableName"`
		EmbeddingTableName         string   `json:"embeddingTableName"`
	} `json:"graph"`
	Languages struct {
		CodeLanguages          []LanguageContract `json:"codeLanguages"`
		RubyExtensionlessFiles []string           `json:"rubyExtensionlessFiles"`
		AuxiliarySyntax        map[string]string  `json:"auxiliarySyntax"`
		AuxiliaryBasenames     map[string]string  `json:"auxiliaryBasenames"`
	} `json:"languages"`
	Pipeline struct {
		Phases []string `json:"phases"`
	} `json:"pipeline"`
	Session struct {
		Providers           []string `json:"providers"`
		Availability        []string `json:"availability"`
		ExecutionModes      []string `json:"executionModes"`
		RuntimeEnvironments []string `json:"runtimeEnvironments"`
		ErrorCodes          []string `json:"errorCodes"`
		StreamEvents        []string `json:"streamEvents"`
	} `json:"session"`
}

var nodeLabels = []scopeir.NodeLabel{
	scopeir.NodeProject,
	scopeir.NodePackage,
	scopeir.NodeModule,
	scopeir.NodeFolder,
	scopeir.NodeFile,
	scopeir.NodeClass,
	scopeir.NodeFunction,
	scopeir.NodeMethod,
	scopeir.NodeVariable,
	scopeir.NodeInterface,
	scopeir.NodeEnum,
	scopeir.NodeDecorator,
	scopeir.NodeImport,
	scopeir.NodeType,
	scopeir.NodeCodeElement,
	scopeir.NodeCommunity,
	scopeir.NodeProcess,
	scopeir.NodeStruct,
	scopeir.NodeMacro,
	scopeir.NodeTypedef,
	scopeir.NodeUnion,
	scopeir.NodeNamespace,
	scopeir.NodeTrait,
	scopeir.NodeImpl,
	scopeir.NodeTypeAlias,
	scopeir.NodeConst,
	scopeir.NodeStatic,
	scopeir.NodeProperty,
	scopeir.NodeRecord,
	scopeir.NodeDelegate,
	scopeir.NodeAnnotation,
	scopeir.NodeConstructor,
	scopeir.NodeTemplate,
	scopeir.NodeSection,
	scopeir.NodeRoute,
	scopeir.NodeTool,
}

var graphRelationshipTypes = []graph.RelationshipType{
	graph.RelContains,
	graph.RelCalls,
	graph.RelInherits,
	graph.RelMethodOverrides,
	graph.RelMethodImplements,
	graph.RelImports,
	graph.RelUses,
	graph.RelDefines,
	graph.RelDecorates,
	graph.RelImplements,
	graph.RelExtends,
	graph.RelHasMethod,
	graph.RelHasProperty,
	graph.RelAccesses,
	graph.RelMemberOf,
	graph.RelStepInProcess,
	graph.RelHandlesRoute,
	graph.RelFetches,
	graph.RelHandlesTool,
	graph.RelEntryPointOf,
	graph.RelWraps,
	graph.RelQueries,
}

var codeLanguages = []LanguageContract{
	{Name: "JavaScript", Value: string(scanner.JavaScript), Extensions: []string{".js", ".jsx", ".mjs", ".cjs"}, Syntax: "javascript"},
	{Name: "TypeScript", Value: string(scanner.TypeScript), Extensions: []string{".ts", ".tsx", ".mts", ".cts"}, Syntax: "typescript"},
	{Name: "Python", Value: string(scanner.Python), Extensions: []string{".py"}, Syntax: "python"},
	{Name: "Java", Value: string(scanner.Java), Extensions: []string{".java"}, Syntax: "java"},
	{Name: "C", Value: string(scanner.C), Extensions: []string{".c"}, Syntax: "c"},
	{Name: "CPlusPlus", Value: string(scanner.CPlusPlus), Extensions: []string{".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx", ".hh"}, Syntax: "cpp"},
	{Name: "CSharp", Value: string(scanner.CSharp), Extensions: []string{".cs"}, Syntax: "csharp"},
	{Name: "Go", Value: string(scanner.Go), Extensions: []string{".go"}, Syntax: "go"},
	{Name: "Ruby", Value: string(scanner.Ruby), Extensions: []string{".rb", ".rake", ".gemspec"}, Syntax: "ruby"},
	{Name: "Rust", Value: string(scanner.Rust), Extensions: []string{".rs"}, Syntax: "rust"},
	{Name: "PHP", Value: string(scanner.PHP), Extensions: []string{".php", ".phtml", ".php3", ".php4", ".php5", ".php8"}, Syntax: "php"},
	{Name: "Kotlin", Value: string(scanner.Kotlin), Extensions: []string{".kt", ".kts"}, Syntax: "kotlin"},
	{Name: "Swift", Value: string(scanner.Swift), Extensions: []string{".swift"}, Syntax: "swift"},
	{Name: "Dart", Value: string(scanner.Dart), Extensions: []string{".dart"}, Syntax: "dart"},
	{Name: "Vue", Value: string(scanner.Vue), Extensions: []string{".vue"}, Syntax: "typescript"},
	{Name: "Svelte", Value: string(scanner.Svelte), Extensions: []string{".svelte"}, Syntax: "typescript"},
	{Name: "Astro", Value: string(scanner.Astro), Extensions: []string{".astro"}, Syntax: "typescript"},
	{Name: "Cobol", Value: string(scanner.Cobol), Extensions: []string{".cbl", ".cob", ".cpy", ".cobol", ".copybook", ".jcl", ".job", ".proc"}, Syntax: "cobol"},
}

var rubyExtensionlessFiles = []string{
	"Rakefile",
	"Gemfile",
	"Guardfile",
	"Vagrantfile",
	"Brewfile",
}

var auxiliarySyntax = map[string]string{
	"json":       "json",
	"yaml":       "yaml",
	"yml":        "yaml",
	"md":         "markdown",
	"mdx":        "markdown",
	"html":       "markup",
	"htm":        "markup",
	"erb":        "markup",
	"xml":        "markup",
	"css":        "css",
	"scss":       "css",
	"sass":       "css",
	"sh":         "bash",
	"bash":       "bash",
	"zsh":        "bash",
	"sql":        "sql",
	"toml":       "toml",
	"ini":        "ini",
	"dockerfile": "docker",
}

var auxiliaryBasenames = map[string]string{
	"Makefile":   "makefile",
	"Dockerfile": "docker",
}

var pipelinePhases = []string{
	"idle",
	"extracting",
	"structure",
	"parsing",
	"imports",
	"calls",
	"heritage",
	"communities",
	"processes",
	"enriching",
	"complete",
	"error",
}

var sessionProviders = []string{"codex", "claude-code"}
var sessionAvailability = []string{"ready", "not_installed", "not_signed_in", "error"}
var sessionExecutionModes = []string{"sandboxed", "bypass"}
var sessionRuntimeEnvironments = []string{"native", "wsl2"}
var sessionErrorCodes = []string{
	"BAD_REQUEST",
	"INVALID_REPO_BINDING",
	"INVALID_REPO_PATH",
	"REPO_NOT_FOUND",
	"INDEX_REQUIRED",
	"SESSION_NOT_FOUND",
	"SESSION_RUNTIME_UNAVAILABLE",
	"SESSION_NOT_SIGNED_IN",
	"SESSION_START_FAILED",
	"SESSION_CANCELLED",
}
var sessionStreamEvents = []string{
	"session_started",
	"reasoning",
	"content",
	"tool_call",
	"tool_result",
	"error",
	"cancelled",
	"done",
}

func WebUIContract() WebUIContractManifest {
	var manifest WebUIContractManifest
	manifest.Status = "go_owned_web_contract_generated"
	manifest.GeneratedFrom = "internal/contracts"
	manifest.Artifacts.SchemaManifest = WebUIContractSchemaPath
	manifest.Artifacts.TypeScriptAdapter = WebUIContractTypeScriptPath
	manifest.Graph.NodeLabels = labelStrings(nodeLabels)
	manifest.Graph.GraphRelationshipTypes = relationshipStrings(graphRelationshipTypes)
	manifest.Graph.LadybugDBNodeTables = append([]string(nil), lbugschema.NodeTables...)
	manifest.Graph.LadybugDBRelationshipTypes = append([]string(nil), lbugschema.RelationshipTypes...)
	manifest.Graph.RelationshipTableName = lbugschema.RelTableName
	manifest.Graph.EmbeddingTableName = lbugschema.EmbeddingTableName
	manifest.Languages.CodeLanguages = append([]LanguageContract(nil), codeLanguages...)
	manifest.Languages.RubyExtensionlessFiles = append([]string(nil), rubyExtensionlessFiles...)
	manifest.Languages.AuxiliarySyntax = copyStringMap(auxiliarySyntax)
	manifest.Languages.AuxiliaryBasenames = copyStringMap(auxiliaryBasenames)
	manifest.Pipeline.Phases = append([]string(nil), pipelinePhases...)
	manifest.Session.Providers = append([]string(nil), sessionProviders...)
	manifest.Session.Availability = append([]string(nil), sessionAvailability...)
	manifest.Session.ExecutionModes = append([]string(nil), sessionExecutionModes...)
	manifest.Session.RuntimeEnvironments = append([]string(nil), sessionRuntimeEnvironments...)
	manifest.Session.ErrorCodes = append([]string(nil), sessionErrorCodes...)
	manifest.Session.StreamEvents = append([]string(nil), sessionStreamEvents...)
	return manifest
}

func WebUIContractJSON() ([]byte, error) {
	raw, err := json.MarshalIndent(WebUIContract(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func WebUIContractTypeScript() (string, error) {
	manifest := WebUIContract()
	var b strings.Builder
	b.WriteString("/* eslint-disable */\n")
	b.WriteString("// Code generated by `go run ./cmd/generate-web-contracts`; DO NOT EDIT.\n")
	b.WriteString("// Source of truth: internal/contracts and Go runtime contract packages.\n\n")
	writeConstArray(&b, "NODE_LABELS", manifest.Graph.NodeLabels)
	b.WriteString("export type NodeLabel = (typeof NODE_LABELS)[number];\n\n")
	writeConstArray(&b, "GRAPH_RELATIONSHIP_TYPES", manifest.Graph.GraphRelationshipTypes)
	b.WriteString("export type RelationshipType = (typeof GRAPH_RELATIONSHIP_TYPES)[number];\n\n")
	writeConstArray(&b, "NODE_TABLES", manifest.Graph.LadybugDBNodeTables)
	b.WriteString("export type NodeTableName = (typeof NODE_TABLES)[number];\n\n")
	writeConstArray(&b, "REL_TYPES", manifest.Graph.LadybugDBRelationshipTypes)
	b.WriteString("export type RelType = (typeof REL_TYPES)[number];\n\n")
	writeConstString(&b, "REL_TABLE_NAME", manifest.Graph.RelationshipTableName)
	writeConstString(&b, "EMBEDDING_TABLE_NAME", manifest.Graph.EmbeddingTableName)
	b.WriteString("\n")
	if err := writeLanguageSection(&b, manifest.Languages.CodeLanguages); err != nil {
		return "", err
	}
	writeConstArray(&b, "RUBY_EXTENSIONLESS_FILES", manifest.Languages.RubyExtensionlessFiles)
	writeConstObject(&b, "AUXILIARY_SYNTAX_MAP", manifest.Languages.AuxiliarySyntax)
	writeConstObject(&b, "AUXILIARY_BASENAME_MAP", manifest.Languages.AuxiliaryBasenames)
	b.WriteString(languageFunctions)
	b.WriteString(graphTypes)
	writeConstArray(&b, "PIPELINE_PHASES", manifest.Pipeline.Phases)
	b.WriteString("export type PipelinePhase = (typeof PIPELINE_PHASES)[number];\n\n")
	b.WriteString(pipelineTypes)
	writeConstArray(&b, "SESSION_PROVIDERS", manifest.Session.Providers)
	b.WriteString("export type LocalSessionProvider = (typeof SESSION_PROVIDERS)[number];\n\n")
	writeConstArray(&b, "SESSION_AVAILABILITY", manifest.Session.Availability)
	b.WriteString("export type SessionAvailability = (typeof SESSION_AVAILABILITY)[number];\n\n")
	writeConstArray(&b, "SESSION_EXECUTION_MODES", manifest.Session.ExecutionModes)
	b.WriteString("export type SessionExecutionMode = (typeof SESSION_EXECUTION_MODES)[number];\n\n")
	writeConstArray(&b, "SESSION_RUNTIME_ENVIRONMENTS", manifest.Session.RuntimeEnvironments)
	b.WriteString("export type SessionRuntimeEnvironment = (typeof SESSION_RUNTIME_ENVIRONMENTS)[number];\n\n")
	writeConstArray(&b, "SESSION_ERROR_CODES", manifest.Session.ErrorCodes)
	b.WriteString("export type SessionErrorCode = (typeof SESSION_ERROR_CODES)[number];\n\n")
	writeConstArray(&b, "SESSION_STREAM_EVENTS", manifest.Session.StreamEvents)
	b.WriteString("export type SessionStreamEventType = (typeof SESSION_STREAM_EVENTS)[number];\n\n")
	b.WriteString(sessionTypes)
	return b.String(), nil
}

func labelStrings(labels []scopeir.NodeLabel) []string {
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		out = append(out, string(label))
	}
	return out
}

func relationshipStrings(types []graph.RelationshipType) []string {
	out := make([]string, 0, len(types))
	for _, relType := range types {
		out = append(out, string(relType))
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func writeConstString(b *strings.Builder, name string, value string) {
	raw, _ := json.Marshal(value)
	fmt.Fprintf(b, "export const %s = %s as const;\n", name, raw)
}

func writeConstArray(b *strings.Builder, name string, values []string) {
	raw, _ := json.MarshalIndent(values, "", "  ")
	fmt.Fprintf(b, "export const %s = %s as const;\n\n", name, string(raw))
}

func writeConstObject(b *strings.Builder, name string, values map[string]string) {
	raw, _ := json.MarshalIndent(values, "", "  ")
	fmt.Fprintf(b, "const %s: Record<string, string> = %s;\n\n", name, string(raw))
}

func writeLanguageSection(b *strings.Builder, languages []LanguageContract) error {
	b.WriteString("export const SupportedLanguages = {\n")
	for _, lang := range languages {
		raw, _ := json.Marshal(lang.Value)
		fmt.Fprintf(b, "  %s: %s,\n", lang.Name, raw)
	}
	b.WriteString("} as const;\n")
	b.WriteString("export type SupportedLanguages = (typeof SupportedLanguages)[keyof typeof SupportedLanguages];\n\n")

	extensionMap := make(map[string][]string, len(languages))
	syntaxMap := make(map[string]string, len(languages))
	for _, lang := range languages {
		extensionMap[lang.Value] = lang.Extensions
		syntaxMap[lang.Value] = lang.Syntax
	}
	extensionRaw, err := marshalStable(extensionMap)
	if err != nil {
		return err
	}
	syntaxRaw, err := marshalStable(syntaxMap)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "const EXTENSION_MAP = %s as const satisfies Record<SupportedLanguages, readonly string[]>;\n\n", extensionRaw)
	fmt.Fprintf(b, "const SYNTAX_MAP = %s as const satisfies Record<SupportedLanguages, string>;\n\n", syntaxRaw)
	return nil
}

func marshalStable(value any) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

const languageFunctions = `const extToLang = new Map<string, SupportedLanguages>();
for (const [lang, exts] of Object.entries(EXTENSION_MAP) as [
  SupportedLanguages,
  readonly string[],
][]) {
  for (const ext of exts) {
    extToLang.set(ext, lang);
  }
}

export const getLanguageFromFilename = (filename: string): SupportedLanguages | null => {
  const normalized = filename.replace(/\\/g, '/');
  const lastDot = normalized.lastIndexOf('.');
  if (lastDot >= 0) {
    const ext = normalized.slice(lastDot).toLowerCase();
    const lang = extToLang.get(ext);
    if (lang !== undefined) return lang;
  }

  const basename = normalized.split('/').pop() || normalized;
  if ((RUBY_EXTENSIONLESS_FILES as readonly string[]).includes(basename)) {
    return SupportedLanguages.Ruby;
  }

  return null;
};

export const getSyntaxLanguageFromFilename = (filePath: string): string => {
  const normalized = filePath.replace(/\\/g, '/');
  const lang = getLanguageFromFilename(normalized);
  if (lang) return SYNTAX_MAP[lang];
  const ext = normalized.split('.').pop()?.toLowerCase();
  if (ext && ext in AUXILIARY_SYNTAX_MAP) return AUXILIARY_SYNTAX_MAP[ext];
  const basename = normalized.split('/').pop() || '';
  if (basename in AUXILIARY_BASENAME_MAP) return AUXILIARY_BASENAME_MAP[basename];
  return 'text';
};

`

const graphTypes = `export type NodeProperties = {
  name: string;
  filePath: string;
  startLine?: number;
  endLine?: number;
  language?: SupportedLanguages | string;
  isExported?: boolean;
  astFrameworkMultiplier?: number;
  astFrameworkReason?: string;
  heuristicLabel?: string;
  cohesion?: number;
  symbolCount?: number;
  keywords?: string[];
  description?: string;
  enrichedBy?: 'heuristic' | 'llm';
  processType?: 'intra_community' | 'cross_community';
  stepCount?: number;
  communities?: string[];
  entryPointId?: string;
  terminalId?: string;
  entryPointScore?: number;
  entryPointReason?: string;
  parameterCount?: number;
  level?: number;
  returnType?: string;
  declaredType?: string;
  visibility?: string;
  isStatic?: boolean;
  isReadonly?: boolean;
  isAbstract?: boolean;
  isFinal?: boolean;
  isVirtual?: boolean;
  isOverride?: boolean;
  isAsync?: boolean;
  isPartial?: boolean;
  annotations?: string[];
  responseKeys?: string[];
  errorKeys?: string[];
  middleware?: string[];
  [key: string]: unknown;
};

export interface GraphNode {
  id: string;
  label: NodeLabel;
  properties: NodeProperties;
}

export interface GraphRelationship {
  id: string;
  sourceId: string;
  targetId: string;
  type: RelationshipType;
  confidence: number;
  reason: string;
  step?: number;
  resolutionSource?: string;
  fileHash?: string;
  evidence?: readonly {
    readonly kind: string;
    readonly weight: number;
    readonly note?: string;
  }[];
}

`

const pipelineTypes = `export interface PipelineProgress {
  phase: PipelinePhase;
  percent: number;
  showPercent?: boolean;
  message: string;
  detail?: string;
  targetRepoName?: string;
  stats?: {
    filesProcessed: number;
    totalFiles: number;
    nodesCreated: number;
  };
}

`

const sessionTypes = `export interface SessionRepoBinding {
  repoName?: string;
  repoPath?: string;
}

export interface ResolvedSessionRepo {
  repoName: string;
  repoPath: string;
  indexed: boolean;
  storagePath?: string;
}

export interface SessionRepoResolution extends SessionRepoBinding {
  state: 'indexed' | 'index_required' | 'not_found' | 'invalid';
  resolvedRepoName?: string;
  resolvedRepoPath?: string;
  message?: string;
}

export interface SessionStatus {
  provider: LocalSessionProvider;
  availability: SessionAvailability;
  available: boolean;
  authenticated: boolean;
  executablePath?: string;
  version?: string;
  message?: string;
  recommendedEnvironment?: 'native' | 'wsl2';
  runtimeEnvironment: SessionRuntimeEnvironment;
  executionMode: SessionExecutionMode;
  supportsSse: boolean;
  supportsCancel: boolean;
  supportsMcp: boolean;
}

export interface SessionStatusResponse extends SessionStatus {
  repo?: SessionRepoResolution;
}

export interface SessionChatRequest extends SessionRepoBinding {
  message: string;
}

export interface SessionToolCall {
  id: string;
  name: string;
  args?: Record<string, unknown>;
  result?: string;
  status: 'pending' | 'running' | 'completed' | 'error';
}

interface SessionEventBase {
  sessionId: string;
  provider: LocalSessionProvider;
  repoName: string;
  repoPath: string;
  timestamp: number;
}

export interface SessionStartedEvent extends SessionEventBase {
  type: 'session_started';
  runtimeEnvironment: SessionRuntimeEnvironment;
  executionMode: SessionExecutionMode;
}

export interface SessionReasoningEvent extends SessionEventBase {
  type: 'reasoning';
  reasoning: string;
}

export interface SessionContentEvent extends SessionEventBase {
  type: 'content';
  content: string;
}

export interface SessionToolCallEvent extends SessionEventBase {
  type: 'tool_call';
  toolCall: SessionToolCall;
}

export interface SessionToolResultEvent extends SessionEventBase {
  type: 'tool_result';
  toolCall: SessionToolCall;
}

export interface SessionErrorEvent extends SessionEventBase {
  type: 'error';
  code: SessionErrorCode;
  error: string;
}

export interface SessionCancelledEvent extends SessionEventBase {
  type: 'cancelled';
  reason: string;
}

export interface SessionDoneEvent extends SessionEventBase {
  type: 'done';
  usage?: Record<string, number>;
}

export type SessionStreamEvent =
  | SessionStartedEvent
  | SessionReasoningEvent
  | SessionContentEvent
  | SessionToolCallEvent
  | SessionToolResultEvent
  | SessionErrorEvent
  | SessionCancelledEvent
  | SessionDoneEvent;
`
