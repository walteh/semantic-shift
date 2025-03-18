package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/ast"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	errors "gitlab.com/tozd/go/errors"
	"gopkg.in/yaml.v3"
)

type TaskRegistry struct {
	server      *server.MCPServer
	taskfile    *ast.Taskfile
	filePath    string
	tasksByName map[string]*ast.Task
	toolNames   map[string]string // Maps task names to tool IDs
	mu          sync.RWMutex
}

// loggerMiddleware wraps an HTTP handler with request/response logging
func loggerMiddleware(next http.Handler, logger zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract request details for logging
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		// Create logger with request context
		reqLogger := logger.With().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("content_type", r.Header.Get("Content-Type")).
			Logger()

		// Only log bodies for specific content types
		shouldLogBody := false
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/plain") ||
			strings.Contains(contentType, "text/html") ||
			strings.Contains(contentType, "application/x-www-form-urlencoded") {
			shouldLogBody = true
		}

		// Max size for body logging (8KB)
		const maxBodyLogSize = 8 * 1024

		// Capture request body if appropriate
		var requestBody []byte
		if r.Body != nil && shouldLogBody {
			// Read up to maxBodyLogSize bytes
			requestBody, _ = io.ReadAll(io.LimitReader(r.Body, maxBodyLogSize))
			// Restore the body for further processing
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Log request with body
		reqEvent := reqLogger.Info().Str("phase", "request")
		if len(requestBody) > 0 {
			// Try to pretty-print JSON bodies
			if strings.Contains(contentType, "application/json") {
				var prettyJSON bytes.Buffer
				if json.Indent(&prettyJSON, requestBody, "", "  ") == nil {
					reqEvent.Str("body", prettyJSON.String())
				} else {
					reqEvent.Str("body", string(requestBody))
				}
			} else {
				// For non-JSON, just log as string but limit size
				if len(requestBody) > maxBodyLogSize {
					reqEvent.Str("body", string(requestBody[:maxBodyLogSize])+"... [truncated]")
				} else {
					reqEvent.Str("body", string(requestBody))
				}
			}
		}
		reqEvent.Msg("Received request")

		// Add request ID to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "requestID", requestID)
		r = r.WithContext(ctx)

		// Create a custom response writer to capture the response
		rw := newLoggingResponseWriter(w)

		// Process the request
		next.ServeHTTP(rw, r)

		// Log the response with body
		duration := time.Since(start)
		respEvent := reqLogger.Info().
			Str("phase", "response").
			Int("status", rw.statusCode).
			Int("size", rw.size).
			Str("content_type", rw.Header().Get("Content-Type")).
			Dur("duration", duration)

		// Include response body if captured and not too large
		respContentType := rw.Header().Get("Content-Type")
		shouldLogRespBody := strings.Contains(respContentType, "application/json") ||
			strings.Contains(respContentType, "text/plain") ||
			strings.Contains(respContentType, "text/html")

		if len(rw.body) > 0 && shouldLogRespBody {
			// Limit response body logging
			bodyToLog := rw.body
			if len(bodyToLog) > maxBodyLogSize {
				bodyToLog = bodyToLog[:maxBodyLogSize]
				respEvent.Bool("truncated", true)
			}

			// Try to pretty-print JSON bodies
			if strings.Contains(respContentType, "application/json") {
				var prettyJSON bytes.Buffer
				if json.Indent(&prettyJSON, bodyToLog, "", "  ") == nil {
					respEvent.Str("body", prettyJSON.String())
				} else {
					respEvent.Str("body", string(bodyToLog))
				}
			} else {
				respEvent.Str("body", string(bodyToLog))
			}
		}

		respEvent.Msg("Request completed")
	})
}

// loggingResponseWriter is a custom ResponseWriter that captures the status code, size and body
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	size        int
	body        []byte
	bodyCapture bool
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200 OK
		bodyCapture:    true,          // Enable body capture by default
	}
}

// WriteHeader implements http.ResponseWriter
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code

	// Decide if we should capture the body based on content type and size
	contentType := lrw.Header().Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") ||
		strings.Contains(contentType, "application/octet-stream") {
		// Don't capture bodies for streaming responses
		lrw.bodyCapture = false
	}

	lrw.ResponseWriter.WriteHeader(code)
}

// Write implements http.ResponseWriter
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	// Capture the response body if enabled and not too large
	const maxCaptureSize = 32 * 1024 // 32KB max total capture
	if lrw.bodyCapture && len(lrw.body) < maxCaptureSize {
		bytesToCapture := b
		remainingSpace := maxCaptureSize - len(lrw.body)
		if len(b) > remainingSpace {
			bytesToCapture = b[:remainingSpace]
		}
		lrw.body = append(lrw.body, bytesToCapture...)
	}

	size, err := lrw.ResponseWriter.Write(b)
	lrw.size += size
	return size, err
}

// Flush implements http.Flusher
func (lrw *loggingResponseWriter) Flush() {
	if flusher, ok := lrw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Push implements http.Pusher for HTTP/2 support
func (lrw *loggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := lrw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// CloseNotify implements http.CloseNotifier (deprecated but sometimes still used)
func (lrw *loggingResponseWriter) CloseNotify() <-chan bool {
	if cn, ok := lrw.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	// If the underlying ResponseWriter doesn't implement CloseNotifier, return a never-closing channel
	ch := make(chan bool, 1)
	return ch
}

// Hijack implements http.Hijacker
func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("http.Hijacker not supported")
}

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

func main() {
	// Create context
	ctx := context.Background()

	// Command line flags
	httpMode := flag.Bool("http", false, "Run in HTTP mode instead of stdio")
	httpAddr := flag.String("addr", ":8080", "HTTP server address (only used with -http)")
	taskfilePath := flag.String("taskfile", "", "Path to Taskfile.yaml (default: auto-detect)")
	logFilePath := flag.String("log", "", "Path to log file (default: logs/taskmcp.log)")
	logLevel := flag.String("log-level", "info", "Log level (trace, debug, info, warn, error, fatal, panic)")
	flag.Parse()

	// Set up logging
	logDir := "logs"
	if *logFilePath == "" {
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			os.Mkdir(logDir, 0755)
		}
		*logFilePath = filepath.Join(logDir, "taskmcp.log")
	} else {
		// Ensure directory exists for custom log path
		logFileDir := filepath.Dir(*logFilePath)
		if _, err := os.Stat(logFileDir); os.IsNotExist(err) {
			os.MkdirAll(logFileDir, 0755)
		}
	}

	// Set log level
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		fmt.Printf("Invalid log level '%s', using 'info'\n", *logLevel)
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Create log file
	logFile, err := os.Create(*logFilePath)
	if err != nil {
		zlog.Fatal().Err(err).Msg("Failed to create log file")
	}
	defer logFile.Close()

	// Configure logger to write to both console and file
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	multi := zerolog.MultiLevelWriter(consoleWriter, logFile)

	// Set global logger
	logger := zerolog.New(multi).With().Timestamp().Caller().Logger()
	zlog.Logger = logger

	// Store logger in context
	ctx = logger.WithContext(ctx)

	// Log startup
	logger.Info().
		Str("mode", map[bool]string{true: "http", false: "stdio"}[*httpMode]).
		Str("logFile", *logFilePath).
		Str("logLevel", level.String()).
		Msg("Starting TaskMCP server")

	// Create MCP server
	s := server.NewMCPServer(
		"TaskMCP",
		"1.0.0",
	)

	registry := &TaskRegistry{
		server:      s,
		tasksByName: make(map[string]*ast.Task),
		toolNames:   make(map[string]string),
	}

	// Find and load Taskfile
	var taskFileToLoad string
	if *taskfilePath != "" {
		taskFileToLoad = *taskfilePath
	} else {
		// Auto-detect Taskfile
		path, err := taskfile.ExistsWalk(".")
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to find Taskfile")
		}
		taskFileToLoad = path
	}

	// Load tools from Taskfile
	tools, err := registry.loadTaskfileHandler(ctx, taskFileToLoad, false)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load Taskfile")
	}

	// Register all the tools
	for taskName, tool := range tools {
		taskNameCopy := taskName // Create a copy to avoid closure-related issues
		s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return registry.executeTaskHandler(ctx, req, taskNameCopy)
		})
		logger.Info().Str("task", taskName).Msg("Registered tool for task")
	}

	logger.Info().
		Int("taskCount", len(tools)).
		Str("taskfile", taskFileToLoad).
		Msg("Loaded tasks from Taskfile")

	// Start server based on mode
	if *httpMode {
		// HTTP mode with SSE
		logger.Info().Str("address", *httpAddr).Msg("Starting HTTP server")

		// Create SSE server
		sseServer := server.NewSSEServer(s)

		// Create a custom HTTP server with our logger middleware
		httpServer := &http.Server{
			Addr:    *httpAddr,
			Handler: loggerMiddleware(sseServer, logger),
		}

		// Start the HTTP server
		logger.Info().Str("address", *httpAddr).Msg("Server is ready to accept connections")
		if err := httpServer.ListenAndServe(); err != nil {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	} else {
		// Stdio mode
		logger.Info().Msg("Starting stdio server")

		// Create standard library logger that writes to our zerolog output
		stdLogger := log.New(logFile, "", log.LstdFlags)

		if err := server.ServeStdio(s, server.WithErrorLogger(stdLogger)); err != nil {
			logger.Fatal().Err(err).Msg("Server error")
			os.Exit(1)
		}
	}
}

func (r *TaskRegistry) loadTaskfileHandler(ctx context.Context, filepathd string, watch bool) (map[string]mcp.Tool, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("filepath", filepathd).Msg("Loading Taskfile")

	if filepathd == "" {
		path, err := taskfile.ExistsWalk(".")
		if err != nil {
			return nil, errors.Errorf("failed to find Taskfile: %w", err)
		}
		filepathd = path
		logger.Debug().Str("detected_path", path).Msg("Auto-detected Taskfile path")
	}

	// Make sure the file exists
	absPath, err := filepath.Abs(filepathd)
	if err != nil {
		return nil, errors.Errorf("failed to resolve path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, errors.Errorf("Taskfile not found at %s", absPath)
	}

	logger.Debug().Str("absolute_path", absPath).Msg("Reading Taskfile")

	// Load the taskfile
	result, err := r.loadTaskfileFromPath(ctx, absPath)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *TaskRegistry) loadTaskfileFromPath(ctx context.Context, absPath string) (map[string]mcp.Tool, error) {
	logger := zerolog.Ctx(ctx)

	// Read the taskfile
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, errors.Errorf("reading Taskfile: %w", err)
	}

	logger.Debug().Int("bytes_read", len(data)).Msg("Read Taskfile")

	// Parse the taskfile using raw yaml to extract task names and details
	var taskfileRaw map[string]interface{}
	if err := yaml.Unmarshal(data, &taskfileRaw); err != nil {
		return nil, errors.Errorf("parsing Taskfile: %w", err)
	}

	// Extract tasks from the raw map
	tasksMap, ok := taskfileRaw["tasks"].(map[string]interface{})
	if !ok {
		return nil, errors.New("no tasks found in Taskfile")
	}

	logger.Debug().Int("task_count", len(tasksMap)).Msg("Found tasks in Taskfile")

	// Parse the taskfile to get the AST for detailed task info
	var taskfileData ast.Taskfile
	if err := yaml.Unmarshal(data, &taskfileData); err != nil {
		return nil, errors.Errorf("parsing Taskfile AST: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Store parsed taskfile and path
	r.taskfile = &taskfileData
	r.filePath = absPath

	// Clear and rebuild the task map
	r.tasksByName = make(map[string]*ast.Task)
	r.toolNames = make(map[string]string)

	tools := make(map[string]mcp.Tool)

	// Loop through task names from the raw map
	for taskName := range tasksMap {
		// Try to get task details from AST
		taskData, ok := taskfileData.Tasks.Get(taskName)
		if !ok {
			logger.Warn().Str("task", taskName).Msg("Failed to get AST data for task")
			continue
		}

		// Store task by name
		r.tasksByName[taskName] = taskData

		// Create tool for this task
		tool := r.createTaskAsTool(ctx, taskName, taskData)
		tools[taskName] = tool

		// Store tool ID
		toolID := fmt.Sprintf("task_%s", strings.ReplaceAll(taskName, ":", "_"))
		r.toolNames[taskName] = toolID

		logger.Debug().
			Str("task", taskName).
			Str("tool_id", toolID).
			Str("description", taskData.Desc).
			Msg("Created tool for task")
	}

	logger.Info().
		Int("task_count", len(tools)).
		Str("file_path", absPath).
		Msg("Successfully loaded Taskfile")

	return tools, nil
}

func (r *TaskRegistry) createTaskAsTool(ctx context.Context, taskName string, task *ast.Task) mcp.Tool {
	logger := zerolog.Ctx(ctx)

	// Create a tool for this task
	description := task.Desc
	if description == "" {
		description = fmt.Sprintf("Run task '%s'", taskName)
	}

	toolID := fmt.Sprintf("task_%s", strings.ReplaceAll(taskName, ":", "_")) // Sanitize the task name for MCP

	toolOpts := []mcp.ToolOption{
		mcp.WithDescription(description),
	}

	// Add parameters for vars if any
	if task.Vars != nil && task.Vars.Len() > 0 {
		// Extract vars from the task
		varsFromTask := extractVars(task)

		logger.Debug().
			Str("task", taskName).
			Int("var_count", len(varsFromTask)).
			Msg("Adding variables as parameters")

		for varName := range varsFromTask {
			// Add as optional string parameter
			toolOpts = append(toolOpts, mcp.WithString(
				varName,
				mcp.Description(fmt.Sprintf("Variable '%s' for task '%s'", varName, taskName)),
			))
		}
	}

	// Create the tool with all options
	return mcp.NewTool(toolID, toolOpts...)
}

func (r *TaskRegistry) executeTaskHandler(ctx context.Context, request mcp.CallToolRequest, taskName string) (*mcp.CallToolResult, error) {
	logger := zerolog.Ctx(ctx)

	logger.Info().
		Str("task", taskName).
		Interface("arguments", request.Params.Arguments).
		Msg("Executing task")

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get variables from request that match task vars
	vars := make(map[string]string)
	task, ok := r.tasksByName[taskName]
	if !ok {
		logger.Error().Str("task", taskName).Msg("Task not found")
		return mcp.NewToolResultError(fmt.Sprintf("Task '%s' not found", taskName)), nil
	}

	// Extract variable values from the request
	varsFromTask := extractVars(task)
	for varName := range varsFromTask {
		if val, ok := request.Params.Arguments[varName].(string); ok {
			vars[varName] = val
			logger.Debug().
				Str("task", taskName).
				Str("var", varName).
				Str("value", val).
				Msg("Found variable for task")
		}
	}

	// Simulate task execution
	varStr := ""
	if len(vars) > 0 {
		varJSON, _ := json.Marshal(vars)
		varStr = fmt.Sprintf(" with variables: %s", string(varJSON))
	}

	result := fmt.Sprintf("Would execute task '%s'%s", taskName, varStr)
	logger.Info().
		Str("task", taskName).
		Int("var_count", len(vars)).
		Msg("Task execution simulated")

	return mcp.NewToolResultText(result), nil
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
