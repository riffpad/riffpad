package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

type Tool interface {
	Definition(lang string) openai.Tool
	Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error)
}

var Tools = []Tool{
	&ReadFileTool{},
	&WriteFileTool{},
	&ListFilesTool{},
	&BashExecTool{},
	NewWebSearchTool(),
}

func ToolDefinitions(lang string) []openai.Tool {
	defs := make([]openai.Tool, len(Tools))
	for i, t := range Tools {
		defs[i] = t.Definition(lang)
	}
	return defs
}

func FindTool(name string) Tool {
	for _, t := range Tools {
		if t.Definition("en").Function.Name == name {
			return t
		}
	}
	return nil
}

type ReadFileTool struct{}

func (t *ReadFileTool) Definition(lang string) openai.Tool {
	desc := "Read the contents of a text file in the workspace."
	paramDesc := "Relative path to the file within the workspace"
	if lang == "zh" {
		desc = "读取工作区中文本文件的内容。"
		paramDesc = "文件在工作区内的相对路径"
	}
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_read",
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{
						"type":        "string",
						"description": paramDesc,
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

func (t *WriteFileTool) Definition(lang string) openai.Tool {
	desc := "Create or overwrite a file in the workspace. Parent directories are created automatically."
	pathDesc := "Relative path to the file within the workspace"
	contentDesc := "Full content to write to the file"
	if lang == "zh" {
		desc = "在工作区中创建或覆盖文件。父目录会自动创建。"
		pathDesc = "文件在工作区内的相对路径"
		contentDesc = "要写入文件的完整内容"
	}
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_write",
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{
						"type":        "string",
						"description": pathDesc,
					},
					"content": map[string]string{
						"type":        "string",
						"description": contentDesc,
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

func (t *ListFilesTool) Definition(lang string) openai.Tool {
	desc := "List files and directories in a workspace directory."
	paramDesc := "Relative directory path within the workspace. Use '.' for root."
	if lang == "zh" {
		desc = "列出工作区目录中的文件和文件夹。"
		paramDesc = "工作区内的相对目录路径，使用 '.' 表示根目录"
	}
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_list",
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"directory": map[string]string{
						"type":        "string",
						"description": paramDesc,
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

func (t *BashExecTool) Definition(lang string) openai.Tool {
	desc := "Execute a shell command in the workspace. Use for running scripts, installing packages, or starting servers."
	commandDesc := "Shell command to execute"
	timeoutDesc := "Timeout in seconds. Default 30, max 300. Ignored when background is true."
	backgroundDesc := "If true, run the command in the background with nohup and return immediately. Use this for starting dev servers."
	if lang == "zh" {
		desc = "在工作区中执行 shell 命令。用于运行脚本、安装包或启动服务器。"
		commandDesc = "要执行的 shell 命令"
		timeoutDesc = "超时时间（秒）。默认 30，最大 300。background 为 true 时忽略。"
		backgroundDesc = "如果为 true，则使用 nohup 在后台运行命令并立即返回。启动开发服务器时使用。"
	}
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "bash_exec",
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]string{
						"type":        "string",
						"description": commandDesc,
					},
					"timeout": map[string]string{
						"type":        "integer",
						"description": timeoutDesc,
					},
					"background": map[string]string{
						"type":        "boolean",
						"description": backgroundDesc,
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (t *BashExecTool) Execute(ctx context.Context, box sandbox.Sandbox, toolCallID string, args json.RawMessage) (string, bool, error) {
	var params struct {
		Command    string `json:"command"`
		Timeout    int    `json:"timeout"`
		Background bool   `json:"background"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", true, fmt.Errorf("parse args: %w", err)
	}

	if params.Background {
		logFile := fmt.Sprintf(".riffpad-server-%s.log", toolCallID)
		wrapped := fmt.Sprintf("nohup sh -c %q > %s 2>&1 & echo $!", params.Command, logFile)
		result, err := box.Exec(ctx, wrapped, sandbox.ExecOptions{Timeout: 10 * time.Second})
		if err != nil {
			return "", true, err
		}
		pid := strings.TrimSpace(result.Stdout)
		if result.ExitCode != 0 {
			return fmt.Sprintf("failed to start background process (exit %d): %s", result.ExitCode, result.Stderr), true, nil
		}
		return fmt.Sprintf("Background process started with PID %s. Logs: %s", pid, logFile), false, nil
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
