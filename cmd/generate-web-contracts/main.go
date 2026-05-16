package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tamnguyendinh/avmatrix-go/internal/contracts"
)

func main() {
	schemaPath := flag.String("schema", contracts.WebUIContractSchemaPath, "path to generated Web UI contract manifest")
	typeScriptPath := flag.String("typescript", contracts.WebUIContractTypeScriptPath, "path to generated Web UI TypeScript adapter")
	check := flag.Bool("check", false, "verify generated files are current without writing")
	flag.Parse()

	schema, err := contracts.WebUIContractJSON()
	if err != nil {
		fail(err)
	}
	typeScript, err := contracts.WebUIContractTypeScript()
	if err != nil {
		fail(err)
	}

	if *check {
		if err := checkFile(*schemaPath, schema); err != nil {
			fail(err)
		}
		if err := checkFile(*typeScriptPath, []byte(typeScript)); err != nil {
			fail(err)
		}
		return
	}

	if err := writeFile(*schemaPath, schema); err != nil {
		fail(err)
	}
	if err := writeFile(*typeScriptPath, []byte(typeScript)); err != nil {
		fail(err)
	}
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func checkFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is stale; run go run ./cmd/generate-web-contracts", path)
	}
	return nil
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
