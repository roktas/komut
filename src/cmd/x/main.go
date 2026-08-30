package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/roktas/komut/internal/komut"
)

const hookPreamble = "Komut expanded the user's invocation. Treat the content below as the user's instruction for this turn. Do not interpret the original invocation separately.\n\n"

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
		return nil
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil
	}
	if input.CWD == "" {
		return nil
	}

	event := input.HookEventName
	if event == "" {
		event = "UserPromptSubmit"
	}

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

	var rendered strings.Builder
	if err := dispatch(invocation, input.CWD, &rendered); err != nil {
		return err
	}

	var output hookOutput
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
