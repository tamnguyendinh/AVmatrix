package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugnative"
	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
	"github.com/tamnguyendinh/avmatrix-go/internal/version"
)

const protocolVersion = "2025-11-25"

const maxStdioMessageBytes = 10 * 1024 * 1024

type stdioFraming string

const (
	stdioFramingContentLength stdioFraming = "content-length"
	stdioFramingNewline       stdioFraming = "newline"
)

var supportedProtocolVersions = map[string]bool{
	protocolVersion: true,
	"2025-06-18":    true,
	"2025-03-26":    true,
	"2024-11-05":    true,
	"2024-10-07":    true,
}

func SupportsProtocolVersion(version string) bool {
	return supportedProtocolVersions[version]
}

type Config struct {
	Store          repo.Store
	OpenReadRunner func(path string) (lbugnative.ReadRunner, error)
}

type Server struct {
	store          repo.Store
	openReadRunner func(path string) (lbugnative.ReadRunner, error)
	graphCache     *resourceGraphCache
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type listReposResult struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	IndexedAt  string      `json:"indexedAt"`
	LastCommit string      `json:"lastCommit"`
	Stats      *repo.Stats `json:"stats,omitempty"`
}

func NewServer(config Config) Server {
	if config.Store.HomeDir == "" {
		config.Store = repo.NewEnvStore()
	}
	if config.OpenReadRunner == nil {
		config.OpenReadRunner = lbugnative.OpenReadRunner
	}
	return Server{store: config.Store, openReadRunner: config.OpenReadRunner, graphCache: newResourceGraphCache()}
}

func Serve(ctx context.Context, input io.Reader, output io.Writer, config Config) error {
	server := NewServer(config)
	reader := bufio.NewReader(input)
	silencer := &lbugruntime.StdioSilencer{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		message, framing, err := readMessage(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		if err := silencer.Run(func() error {
			response, ok, err := server.HandleJSONRPC(message)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			return writeRawMessageFramed(output, response, framing)
		}); err != nil {
			return err
		}
	}
}

func (s Server) HandleJSONRPC(raw []byte) ([]byte, bool, error) {
	message := bytes.TrimSpace(raw)
	if len(message) == 0 {
		response := errorResponse(nil, -32600, "Invalid request")
		encoded, err := json.Marshal(response)
		return encoded, true, err
	}

	if message[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(message, &batch); err != nil {
			response := errorResponse(nil, -32700, "Parse error")
			encoded, marshalErr := json.Marshal(response)
			return encoded, true, marshalErr
		}
		if len(batch) == 0 {
			response := errorResponse(nil, -32600, "Invalid request")
			encoded, err := json.Marshal(response)
			return encoded, true, err
		}

		responses := make([]rpcResponse, 0, len(batch))
		for _, item := range batch {
			response, ok := s.handle(item)
			if ok {
				responses = append(responses, response)
			}
		}
		if len(responses) == 0 {
			return nil, false, nil
		}
		encoded, err := json.Marshal(responses)
		return encoded, true, err
	}

	response, ok := s.handle(message)
	if !ok {
		return nil, false, nil
	}
	encoded, err := json.Marshal(response)
	return encoded, true, err
}

func (s Server) handle(raw []byte) (rpcResponse, bool) {
	var request rpcRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return errorResponse(nil, -32700, "Parse error"), true
	}
	if request.JSONRPC != "2.0" || request.Method == "" {
		return errorResponse(request.ID, -32600, "Invalid request"), true
	}
	if request.ID == nil {
		return rpcResponse{}, false
	}

	switch request.Method {
	case "initialize":
		return resultResponse(request.ID, s.initialize(request.Params)), true
	case "ping":
		return resultResponse(request.ID, map[string]any{}), true
	case "tools/list":
		return resultResponse(request.ID, map[string]any{"tools": mcpTools()}), true
	case "tools/call":
		result, err := s.callTool(request.Params)
		if err != nil {
			return errorResponse(request.ID, -32602, err.Error()), true
		}
		return resultResponse(request.ID, result), true
	case "resources/list":
		return resultResponse(request.ID, map[string]any{"resources": resourceDefinitions()}), true
	case "resources/templates/list":
		return resultResponse(request.ID, map[string]any{"resourceTemplates": resourceTemplates()}), true
	case "resources/read":
		result, err := s.readResource(request.Params)
		if err != nil {
			return errorResponse(request.ID, -32602, err.Error()), true
		}
		return resultResponse(request.ID, result), true
	case "prompts/list":
		return resultResponse(request.ID, map[string]any{"prompts": promptDefinitions()}), true
	case "prompts/get":
		result, err := getPrompt(request.Params)
		if err != nil {
			return errorResponse(request.ID, -32602, err.Error()), true
		}
		return resultResponse(request.ID, result), true
	default:
		return errorResponse(request.ID, -32601, "Method not found"), true
	}
}

func (s Server) initialize(raw json.RawMessage) map[string]any {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(raw, &params)
	negotiated := protocolVersion
	if supportedProtocolVersions[params.ProtocolVersion] {
		negotiated = params.ProtocolVersion
	}
	return map[string]any{
		"protocolVersion": negotiated,
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
			"prompts":   map[string]any{},
		},
		"serverInfo": map[string]string{
			"name":    "avmatrix",
			"version": version.Version,
		},
	}
}

func (s Server) callTool(raw json.RawMessage) (map[string]any, error) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("Invalid tool call params: %w", err)
	}
	switch params.Name {
	case "list_repos":
		result, err := s.listRepos()
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "query":
		result, err := s.queryTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "cypher":
		result, err := s.cypherTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "context":
		result, err := s.contextTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "impact":
		result, err := s.impactTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "detect_changes":
		result, err := s.detectChangesTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "rename":
		result, err := s.renameTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "route_map":
		result, err := s.routeMapTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "tool_map":
		result, err := s.toolMapTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "shape_check":
		result, err := s.shapeCheckTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "api_impact":
		result, err := s.apiImpactTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "group_list":
		result, err := s.groupListTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "group_status":
		result, err := s.groupStatusTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "group_sync":
		result, err := s.groupSyncTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "group_contracts":
		result, err := s.groupContractsTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	case "group_query":
		result, err := s.groupQueryTool(params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolTextResult(result, nextStepHint(params.Name, params.Arguments))
	default:
		return nil, fmt.Errorf("Unknown tool: %s", params.Name)
	}
}

func (s Server) listRepos() ([]listReposResult, error) {
	entries, err := s.store.ListRegistered(false)
	if err != nil {
		return nil, err
	}
	results := make([]listReposResult, 0, len(entries))
	for _, entry := range entries {
		results = append(results, listReposResult{
			Name:       entry.Name,
			Path:       entry.Path,
			IndexedAt:  entry.IndexedAt,
			LastCommit: entry.LastCommit,
			Stats:      entry.Stats,
		})
	}
	return results, nil
}

func toolTextResult(result any, hint string) (map[string]any, error) {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []map[string]string{{
			"type": "text",
			"text": string(raw) + hint,
		}},
	}, nil
}

func nextStepHint(toolName string, args map[string]any) string {
	repo := stringArg(args, "repo", "")
	repoParam := ""
	if repo != "" {
		repoParam = `, repo: "` + repo + `"`
	}
	repoPath := firstNonEmptyString(repo, "{name}")

	switch toolName {
	case "list_repos":
		return "\n\n---\n**Next:** READ avmatrix://repo/{name}/context for any repo above to get its overview and check staleness."
	case "query":
		return "\n\n---\n**Next:** To understand a specific symbol in depth, use context({name: \"<symbol_name>\"" + repoParam + "}) to see categorized refs and process participation."
	case "context":
		target := firstNonEmptyString(stringArg(args, "name", ""), "<name>")
		return "\n\n---\n**Next:** If planning changes, use impact({target: \"" + target + "\", direction: \"upstream\"" + repoParam + "}) to check blast radius. To see execution flows, READ avmatrix://repo/" + repoPath + "/processes."
	case "impact":
		return "\n\n---\n**Next:** Review d=1 items first (WILL BREAK). To check affected execution flows, READ avmatrix://repo/" + repoPath + "/processes."
	case "detect_changes":
		return "\n\n---\n**Next:** Review affected processes. Use context() on high-risk changed symbols. READ avmatrix://repo/" + repoPath + "/process/{name} for full execution traces."
	case "rename":
		return "\n\n---\n**Next:** Run detect_changes(" + renameHintRepoArg(repo) + ") to verify no unexpected side effects from the rename."
	case "route_map":
		return "\n\n---\n**Next:** For pre-change route analysis, use api_impact({route: \"<route>\"" + repoParam + "}) once api_impact is available; use impact() on specific handlers for symbol blast radius."
	case "tool_map":
		return "\n\n---\n**Next:** Use context({name: \"<handler_or_tool_symbol>\"" + repoParam + "}) or impact({target: \"<symbol>\", direction: \"upstream\"" + repoParam + "}) before changing tool implementations."
	case "shape_check":
		return "\n\n---\n**Next:** For pre-change route risk, use api_impact({route: \"<route>\"" + repoParam + "}) to combine consumers, shape mismatches, middleware, and flows."
	case "api_impact":
		return "\n\n---\n**Next:** Review direct consumers, mismatches, middleware, and execution flows before editing the route handler."
	case "group_list":
		return "\n\n---\n**Next:** Use group_status({name: \"<group>\"}) before group_sync to check stale indexes and contract registry status."
	case "group_status":
		return "\n\n---\n**Next:** Refresh stale member indexes with analyze before running group_sync when contracts are stale."
	case "group_sync":
		return "\n\n---\n**Next:** Use group_contracts({name: \"<group>\"}) to inspect the written Contract Registry."
	case "group_contracts":
		return "\n\n---\n**Next:** Use group_query({name: \"<group>\", query: \"<flow>\"}) to inspect execution flows across matched repos."
	case "group_query":
		return "\n\n---\n**Next:** Use group_contracts({name: \"<group>\"}) to inspect contracts and cross-links behind cross-repo results."
	case "cypher":
		return "\n\n---\n**Next:** To explore a result symbol, use context({name: \"<name>\"" + repoParam + "}). For schema reference, READ avmatrix://repo/" + repoPath + "/schema."
	default:
		return ""
	}
}

func renameHintRepoArg(repo string) string {
	if repo == "" {
		return ""
	}
	return `{repo: "` + repo + `"}`
}

func resultResponse(id any, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id any, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func readMessage(reader *bufio.Reader) ([]byte, stdioFraming, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
			return nil, "", io.EOF
		}
		return nil, "", err
	}
	for strings.TrimSpace(line) == "" {
		line, err = reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
				return nil, "", io.EOF
			}
			return nil, "", err
		}
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if len(trimmed) > maxStdioMessageBytes {
			return nil, "", fmt.Errorf("read buffer exceeded maximum size (%d bytes)", maxStdioMessageBytes)
		}
		return []byte(trimmed), stdioFramingNewline, nil
	}

	contentLength := -1
	for {
		header := strings.TrimRight(line, "\r\n")
		if header == "" {
			break
		}
		name, value, ok := strings.Cut(header, ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			length, parseErr := strconv.Atoi(strings.TrimSpace(value))
			if parseErr != nil {
				return nil, "", parseErr
			}
			if length > maxStdioMessageBytes {
				return nil, "", fmt.Errorf("Content-Length %d exceeds maximum allowed size (%d bytes)", length, maxStdioMessageBytes)
			}
			contentLength = length
		}
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, "", err
		}
	}
	if contentLength < 0 {
		return nil, "", errors.New("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, "", err
	}
	return body, stdioFramingContentLength, nil
}

func writeMessage(writer io.Writer, response rpcResponse) error {
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return writeRawMessage(writer, raw)
}

func writeRawMessage(writer io.Writer, raw []byte) error {
	return writeRawMessageFramed(writer, raw, stdioFramingContentLength)
}

func writeRawMessageFramed(writer io.Writer, raw []byte, framing stdioFraming) error {
	if framing == stdioFramingNewline {
		if _, err := writer.Write(raw); err != nil {
			return err
		}
		_, err := writer.Write([]byte("\n"))
		return err
	}
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(raw)); err != nil {
		return err
	}
	_, err := writer.Write(raw)
	return err
}
