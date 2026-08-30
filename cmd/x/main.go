package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/roktas/komut/internal/komut"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "x: %v\n", err)
		os.Exit(2)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 1 && args[0] == "--hook" {
		return runHook(stdin, stdout)
	}

	input, err := readInvocation(args, stdin)
	if err != nil {
		return err
	}
	return dispatch(input, "", stdout)
}

func runHook(stdin io.Reader, stdout io.Writer) error {
	var input struct {
		Prompt string `json:"prompt"`
		CWD    string `json:"cwd"`
	}
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil
	}
	if !komut.HasInvocationPrefix(input.Prompt) {
		return nil
	}
	return dispatch(input.Prompt, input.CWD, stdout)
}

func dispatch(input, cwd string, stdout io.Writer) error {
	var err error
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	output, err := komut.Dispatch(input, cwd, home)
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, output)
	return err
}

func readInvocation(args []string, stdin io.Reader) (string, error) {
	switch len(args) {
	case 0:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read invocation: %w", err)
		}
		return string(data), nil
	case 1:
		return args[0], nil
	default:
		return "", fmt.Errorf("expected one invocation argument or stdin")
	}
}
