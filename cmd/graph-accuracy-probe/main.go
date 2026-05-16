package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/graphaccuracy"
)

func main() {
	mode := flag.String("mode", "report", "report or enforce")
	repo := flag.String("repo", ".", "repository root")
	nodeGraphPath := flag.String("node", "", "Node/MCP API graph JSON")
	goGraphPath := flag.String("go", "", "Go local API graph JSON")
	outPath := flag.String("out", "", "output JSON path")
	avmatrixPath := flag.String("avmatrix", "", "local AVmatrix binary used to regenerate the Go graph before comparison")
	freshGoGraphPath := flag.String("fresh-go-graph", "", "copy the freshly analyzed Go graph snapshot to this path and use it as -go")
	benchmarkPath := flag.String("benchmark-json", "", "write analyze benchmark metrics JSON when -avmatrix is used")
	benchmarkLabel := flag.String("benchmark-label", "", "attach a label to the analyze benchmark JSON when -avmatrix is used")
	maxExamples := flag.Int("max-examples", 50, "maximum missing/extra examples to include per gate")
	flag.Parse()

	if *nodeGraphPath == "" || *outPath == "" {
		exitf("missing -node or -out")
	}
	if *mode != "report" && *mode != "enforce" {
		exitf("-mode must be report or enforce")
	}
	if *avmatrixPath != "" || *freshGoGraphPath != "" {
		if *avmatrixPath == "" || *freshGoGraphPath == "" {
			exitf("-avmatrix and -fresh-go-graph must be provided together")
		}
		freshGraph, err := runFreshAnalyze(*repo, *avmatrixPath, *freshGoGraphPath, *benchmarkPath, *benchmarkLabel)
		if err != nil {
			exitf("%v", err)
		}
		*goGraphPath = freshGraph
	}
	if *goGraphPath == "" {
		exitf("missing -go, or provide -avmatrix with -fresh-go-graph")
	}

	result, err := graphaccuracy.Run(graphaccuracy.Options{
		Repo:          *repo,
		NodeGraphPath: *nodeGraphPath,
		GoGraphPath:   *goGraphPath,
		OutPath:       *outPath,
		MaxExamples:   *maxExamples,
	})
	if err != nil {
		exitf("%v", err)
	}

	fmt.Printf("wrote %s\n", *outPath)
	for _, line := range graphaccuracy.SummaryLines(result) {
		fmt.Println(line)
	}

	failures := graphaccuracy.GoLocalFailures(result)
	if len(failures) == 0 {
		fmt.Println("accuracy gate: PASS")
		return
	}
	fmt.Printf("accuracy gate: %d failure(s)\n", len(failures))
	for _, failure := range failures {
		fmt.Printf("- %s: %d/%d recall=%.2f", failure.Gate, failure.Matched, failure.Expected, failure.RecallPct)
		if failure.PrecisionPct != 0 {
			fmt.Printf(" precision=%.2f", failure.PrecisionPct)
		}
		fmt.Println()
	}
	if *mode == "enforce" {
		raw, err := json.MarshalIndent(failures, "", "  ")
		if err == nil {
			_, _ = os.Stderr.Write(raw)
			_, _ = os.Stderr.Write([]byte("\n"))
		}
		os.Exit(1)
	}
}

func runFreshAnalyze(repoPath string, avmatrixPath string, graphOutPath string, benchmarkPath string, benchmarkLabel string) (string, error) {
	repoAbs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo: %w", err)
	}
	args := []string{"analyze", repoAbs, "--force", "--skip-agents-md", "--no-stats"}
	if benchmarkPath != "" {
		args = append(args, "--benchmark-json", benchmarkPath)
	}
	if benchmarkLabel != "" {
		args = append(args, "--benchmark-label", benchmarkLabel)
	}
	cmd := exec.Command(avmatrixPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_, _ = io.Copy(os.Stdout, &stdout)
		_, _ = io.Copy(os.Stderr, &stderr)
		return "", fmt.Errorf("run fresh analyze: %w", err)
	}
	_, _ = os.Stdout.Write(stdout.Bytes())
	_, _ = os.Stderr.Write(stderr.Bytes())

	graphPath := parseGraphPath(stdout.String())
	if graphPath == "" {
		graphPath = filepath.Join(repoAbs, ".avmatrix", "graph.json")
	}
	if err := copyFile(graphPath, graphOutPath); err != nil {
		return "", err
	}
	return graphOutPath, nil
}

func parseGraphPath(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.LastIndex(line, "path="); idx >= 0 {
			return strings.TrimSpace(line[idx+len("path="):])
		}
	}
	return ""
}

func copyFile(src string, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read fresh graph %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return fmt.Errorf("write fresh graph copy %s: %w", dst, err)
	}
	return nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
