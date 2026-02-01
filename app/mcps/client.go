package mcps

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"GoWorkerAI/app/tools"
)

type Client struct {
	name   string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu      sync.Mutex
	msgID   int
	pending map[int]chan *Response

	// Tools exposed by this MCP
	tools map[string]tools.Tool
}

type Config struct {
	Name    string            `json:"name" yaml:"name"`
	Command string            `json:"command" yaml:"command"`
	Args    []string          `json:"args" yaml:"args"`
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ListToolsResult struct {
	Tools []MCPTool `json:"tools"`
}

type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)

	// Add custom env vars
	cmd.Env = append(os.Environ(), envMapToSlice(cfg.Env)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start MCP server: %w", err)
	}

	client := &Client{
		name:    cfg.Name,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		pending: make(map[int]chan *Response),
		tools:   make(map[string]tools.Tool),
	}

	go client.readLoop()
	go client.logStderr()

	if err = client.initialize(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err = client.discoverTools(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("discover tools: %w", err)
	}

	log.Printf("✅ MCP Server '%s' started with %d tools\n", cfg.Name, len(client.tools))
	return client, nil
}

func (c *Client) initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "GoWorkerAI",
			"version": "1.0.0",
		},
	}

	_, err := c.call(ctx, "initialize", params)
	if err != nil {
		return err
	}

	return c.notify("notifications/initialized", nil)
}

func (c *Client) discoverTools(ctx context.Context) error {
	resp, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return err
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	for _, mcpTool := range result.Tools {
		c.tools[mcpTool.Name] = c.wrapMCPTool(mcpTool)
	}

	return nil
}

func (c *Client) wrapMCPTool(mcpTool MCPTool) tools.Tool {
	return tools.Tool{
		Name:        fmt.Sprintf("%s/%s", c.name, mcpTool.Name),
		Description: mcpTool.Description,
		Parameters:  c.convertSchema(mcpTool.InputSchema),
		HandlerFunc: func(task tools.ToolTask) (string, error) {
			return c.callTool(context.Background(), mcpTool.Name, task.Parameters)
		},
	}
}

func (c *Client) convertSchema(schema map[string]any) tools.Parameter {
	param := tools.Parameter{
		Type:       "object",
		Properties: make(map[string]any),
		Required:   []string{},
	}

	if props, ok := schema["properties"].(map[string]any); ok {
		param.Properties = props
	}

	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				param.Required = append(param.Required, s)
			}
		}
	}

	return param
}

func (c *Client) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	params := map[string]any{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return "", err
	}

	var result CallToolResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	var output string
	for _, content := range result.Content {
		if content.Type == "text" {
			output += content.Text
		}
	}

	return output, nil
}

func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	respChan := make(chan *Response, 1)
	c.pending[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := json.NewEncoder(c.stdin).Encode(req); err != nil {
		return nil, err
	}

	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) notify(method string, params any) error {
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return json.NewEncoder(c.stdin).Encode(req)
}

func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			log.Printf("⚠️ MCP '%s' invalid response: %v\n", c.name, err)
			continue
		}

		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- &resp
		}
		c.mu.Unlock()
	}
}

func (c *Client) logStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		log.Printf("🔧 MCP '%s': %s\n", c.name, scanner.Text())
	}
}

func (c *Client) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		return c.cmd.Wait()
	}
	return nil
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Tools() map[string]tools.Tool {
	result := make(map[string]tools.Tool, len(c.tools))
	for k, v := range c.tools {
		result[k] = v
	}
	return result
}

func envMapToSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
