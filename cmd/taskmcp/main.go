package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
)

type TaskRegistry struct {
	server      *server.MCPServer
	taskfile    *ast.Taskfile
	filePath    string
	tasksByName map[string]*ast.Task
}

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"TaskMCP",
		"1.0.0",
	)

	registry := &TaskRegistry{
		server:      s,
		tasksByName: make(map[string]*ast.Task),
	}

	// Add task file tool
	taskFileTool := mcp.NewTool("load_taskfile",
		mcp.WithDescription("Load tasks from a Taskfile.yaml"),
		mcp.WithString("filepath",
			mcp.Description("Path to the Taskfile.yaml, defaults to Taskfile.yaml in current directory"),
			mcp.DefaultString("Taskfile.yaml"),
		),
	)

	// Add tool handler
	s.AddTool(taskFileTool, registry.loadTaskfileHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func (r *TaskRegistry) loadTaskfileHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskfilePath, ok := request.Params.Arguments["filepath"].(string)
	if !ok {
		taskfilePath = "Taskfile.yaml"
	}

	// Make sure the file exists
	absPath, err := filepath.Abs(taskfilePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve path: %v", err)), nil
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("Taskfile not found at %s", absPath)), nil
	}

	// Read the taskfile
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read Taskfile: %v", err)), nil
	}

	// Parse the taskfile using raw yaml to extract task names and details
	var taskfileRaw map[string]interface{}
	if err := yaml.Unmarshal(data, &taskfileRaw); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse Taskfile: %v", err)), nil
	}

	// Extract tasks from the raw map
	tasksMap, ok := taskfileRaw["tasks"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("No tasks found in Taskfile"), nil
	}

	// Extract tasks and build response
	tasks := make(map[string]map[string]interface{})
	r.tasksByName = make(map[string]*ast.Task)

	// Parse the taskfile to get the AST for detailed task info
	var taskfileData ast.Taskfile
	if err := yaml.Unmarshal(data, &taskfileData); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse Taskfile AST: %v", err)), nil
	}

	// Store parsed taskfile and path
	r.taskfile = &taskfileData
	r.filePath = absPath

	// Loop through task names from the raw map
	for taskName := range tasksMap {
		// Try to get task details from AST
		taskData, ok := taskfileData.Tasks.Get(taskName)
		if !ok {
			continue
		}

		// Store task by name
		r.tasksByName[taskName] = taskData

		// Extract info for the response
		taskInfo := map[string]interface{}{
			"description": taskData.Desc,
			"commands":    extractCommands(taskData),
			"deps":        extractDeps(taskData),
			"vars":        extractVars(taskData),
		}
		tasks[taskName] = taskInfo

		// Register this task as an MCP tool
		r.registerTaskAsTool(taskName, taskData)
	}

	// Convert to JSON and create result
	result := map[string]interface{}{
		"tasks":        tasks,
		"taskfilePath": absPath,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (r *TaskRegistry) registerTaskAsTool(taskName string, task *ast.Task) {
	// Create a tool for this task
	description := task.Desc
	if description == "" {
		description = fmt.Sprintf("Run task '%s'", taskName)
	}

	toolOpts := []mcp.ToolOption{
		mcp.WithDescription(description),
	}

	// Add parameters for vars if any
	if task.Vars != nil && task.Vars.Len() > 0 {
		// Extract vars from the task
		varsFromTask := extractVars(task)

		for varName := range varsFromTask {
			// Add as optional string parameter
			toolOpts = append(toolOpts, mcp.WithString(
				varName,
				mcp.Description(fmt.Sprintf("Variable '%s' for task '%s'", varName, taskName)),
			))
		}
	}

	// Create the tool with all options
	tool := mcp.NewTool(
		fmt.Sprintf("task_%s", strings.ReplaceAll(taskName, ":", "_")), // Sanitize the task name for MCP
		toolOpts...,
	)

	// Add the task handler
	r.server.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return r.executeTaskHandler(ctx, req, taskName)
	})
}

func (r *TaskRegistry) executeTaskHandler(ctx context.Context, request mcp.CallToolRequest, taskName string) (*mcp.CallToolResult, error) {
	// Here we would implement actual task execution logic using the task package
	// For now, we'll just return a message saying which task would be executed

	// Get variables from request that match task vars
	vars := make(map[string]string)
	task, ok := r.tasksByName[taskName]
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("Task '%s' not found", taskName)), nil
	}

	// Extract variable values from the request
	varsFromTask := extractVars(task)
	for varName := range varsFromTask {
		if val, ok := request.Params.Arguments[varName].(string); ok {
			vars[varName] = val
		}
	}

	// Simulate task execution
	varStr := ""
	if len(vars) > 0 {
		varJSON, _ := json.Marshal(vars)
		varStr = fmt.Sprintf(" with variables: %s", string(varJSON))
	}

	return mcp.NewToolResultText(fmt.Sprintf("Would execute task '%s'%s", taskName, varStr)), nil
}

func extractCommands(task *ast.Task) []string {
	commands := []string{}
	for _, cmd := range task.Cmds {
		if cmd.Cmd != "" {
			commands = append(commands, cmd.Cmd)
		}
	}
	return commands
}

func extractDeps(task *ast.Task) []string {
	deps := []string{}
	for _, dep := range task.Deps {
		deps = append(deps, dep.Task)
	}
	return deps
}

func extractVars(task *ast.Task) map[string]string {
	if task.Vars == nil {
		return make(map[string]string)
	}

	vars := make(map[string]string)

	// Use custom approach to work around iterator issues
	// Convert to YAML and back to map to get the var names
	data, err := yaml.Marshal(task.Vars)
	if err == nil {
		var varsMap map[string]interface{}
		if err := yaml.Unmarshal(data, &varsMap); err == nil {
			for k, v := range varsMap {
				vars[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return vars
}
