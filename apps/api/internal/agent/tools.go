package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

type Tool interface {
	Definition() openai.Tool
	Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error)
}

var Tools = []Tool{
	&ReadFileTool{},
	&WriteFileTool{},
	&ListFilesTool{},
	&BashExecTool{},
}

func ToolDefinitions() []openai.Tool {
	defs := make([]openai.Tool, len(Tools))
	for i, t := range Tools {
		defs[i] = t.Definition()
	}
	return defs
}

func FindTool(name string) Tool {
	for _, t := range Tools {
		if t.Definition().Function.Name == name {
			return t
		}
	}
	return nil
}

type ReadFileTool struct{}

func (t *ReadFileTool) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_read",
			Description: "Read the contents of a text file in the workspace.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{
						"type":        "string",
						"description": "Relative path to the file within the workspace",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", true, fmt.Errorf("parse args: %w", err)
	}

	content, err := box.ReadFile(ctx, params.Path)
	if err != nil {
		return "", true, err
	}
	return content, false, nil
}

type WriteFileTool struct{}

func (t *WriteFileTool) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_write",
			Description: "Create or overwrite a file in the workspace. Parent directories are created automatically.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{
						"type":        "string",
						"description": "Relative path to the file within the workspace",
					},
					"content": map[string]string{
						"type":        "string",
						"description": "Full content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", true, fmt.Errorf("parse args: %w", err)
	}

	if err := box.WriteFile(ctx, params.Path, params.Content); err != nil {
		return "", true, err
	}
	return fmt.Sprintf("File written: %s", params.Path), false, nil
}

type ListFilesTool struct{}

func (t *ListFilesTool) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_list",
			Description: "List files and directories in a workspace directory.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"directory": map[string]string{
						"type":        "string",
						"description": "Relative directory path within the workspace. Use '.' for root.",
					},
				},
				"required": []string{"directory"},
			},
		},
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error) {
	var params struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", true, fmt.Errorf("parse args: %w", err)
	}

	files, err := box.ListFiles(ctx, params.Directory)
	if err != nil {
		return "", true, err
	}

	data, err := json.Marshal(files)
	if err != nil {
		return "", true, err
	}
	return string(data), false, nil
}

type BashExecTool struct{}

func (t *BashExecTool) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "bash_exec",
			Description: "Execute a shell command in the workspace. Use for running scripts, installing packages, or starting servers.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]string{
						"type":        "string",
						"description": "Shell command to execute",
					},
					"timeout": map[string]string{
						"type":        "integer",
						"description": "Timeout in seconds. Default 30, max 300.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (t *BashExecTool) Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error) {
	var params struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", true, fmt.Errorf("parse args: %w", err)
	}

	timeout := sandbox.ExecOptions{Timeout: durationFromSeconds(params.Timeout, 30)}
	result, err := box.Exec(ctx, params.Command, timeout)
	if err != nil {
		return "", true, err
	}

	output := result.Stdout
	if result.Stderr != "" {
		output += "\n[stderr]\n" + result.Stderr
	}

	isError := result.ExitCode != 0
	if isError {
		output = fmt.Sprintf("exit code %d\n%s", result.ExitCode, output)
	}
	return output, isError, nil
}

func durationFromSeconds(seconds, defaultVal int) time.Duration {
	if seconds <= 0 {
		seconds = defaultVal
	}
	if seconds > 300 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}
