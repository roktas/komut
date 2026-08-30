package main

import (
	"fmt"
	"io"
	"os"

	"github.com/roktas/komut/internal/komut"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "x: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	input, err := readInvocation(os.Args[1:], os.Stdin)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	output, err := komut.Dispatch(input, cwd, home)
	if err != nil {
		return err
	}
	_, err = io.WriteString(os.Stdout, output)
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
