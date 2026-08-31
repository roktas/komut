package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/roktas/komut/internal/komut"
)

const hookPreamble = "Use only this expansion of the user's $x invocation:\n\n"

type hookInput struct {
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
	CWD           string `json:"cwd"`
	ExpansionType string `json:"expansion_type"`
	CommandName   string `json:"command_name"`
	CommandArgs   string `json:"command_args"`
	CommandSource string `json:"command_source"`
}

type hookOutput struct {
	SuppressOutput     bool `json:"suppressOutput,omitempty"`
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

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
	var input hookInput
	decoder := json.NewDecoder(stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("decode hook input: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("decode hook input: expected one JSON value")
	}

	event := cmp.Or(input.HookEventName, "UserPromptSubmit")

	invocation := ""
	switch event {
	case "UserPromptSubmit":
		if !komut.HasInvocationPrefix(input.Prompt) {
			return nil
		}
		invocation = input.Prompt
	case "UserPromptExpansion":
		if input.ExpansionType != "slash_command" || input.CommandSource != "plugin" {
			return nil
		}
		if input.CommandName != "komut:x" && input.CommandName != "x" {
			return nil
		}
		invocation = "$x"
		if input.CommandArgs != "" {
			invocation += " " + input.CommandArgs
		}
	default:
		return nil
	}
	if input.CWD == "" {
		return fmt.Errorf("hook input is missing cwd")
	}

	var rendered strings.Builder
	if err := dispatch(invocation, input.CWD, &rendered); err != nil {
		return err
	}

	var output hookOutput
	output.SuppressOutput = event == "UserPromptSubmit"
	output.HookSpecificOutput.HookEventName = event
	output.HookSpecificOutput.AdditionalContext = hookPreamble + rendered.String()
	return json.NewEncoder(stdout).Encode(output)
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
