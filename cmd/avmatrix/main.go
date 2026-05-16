package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/tamnguyendinh/avmatrix-go/internal/cli"
)

func main() {
	root := cli.NewRootCommand(cli.Options{})
	if err := root.ExecuteContext(context.Background()); err != nil {
		var exitErr cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
