# MCP (Model Context Protocol) Support

GoWorkerAI supports the Model Context Protocol (MCP), allowing you to extend functionality through external processes written in any language.

## What is MCP?

MCP is a standardized protocol for connecting AI applications to external data sources and tools. It's similar to LSP (Language Server Protocol) but for AI agents.

## Architecture

```
┌─────────────────┐
│   GoWorkerAI    │
│   (MCP Client)  │
└────────┬────────┘
         │ JSON-RPC over stdio
         ├─────────────┐
         │             │
    ┌────▼────┐   ┌───▼────┐
    │MCP      │   │MCP     │
    │Server   │   │Server  │
    │(Python) │   │(Node)  │
    └────┬────┘   └───┬────┘
         │            │
    ┌────▼────┐  ┌───▼────┐
    │Database │  │GitHub  │
    │Tools    │  │API     │
    └─────────┘  └────────┘
```

## Features

- ✅ JSON-RPC 2.0 over stdio transport
- ✅ Automatic tool discovery
- ✅ Schema conversion (MCP → GoWorkerAI tools)
- ✅ Per-worker MCP configuration
- ✅ Global MCPs for all workers
- ✅ Process lifecycle management
- ✅ Error handling and logging

## Configuration

MCPs are configured in `config.yaml`:

### Global MCPs

Available to all workers in all teams:

```yaml
global_mcps:
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "./workspace"]
    env:
      ALLOWED_DIRECTORIES: "./workspace"
```

### Worker-specific MCPs

Available only to specific workers:

```yaml
teams:
  default:
    members:
      - key: database-admin
        worker_type: coder
        mcps:
          - name: postgres
            command: npx
            args: ["-y", "@modelcontextprotocol/server-postgres"]
            env:
              DATABASE_URL: "postgresql://localhost:5432/mydb"
```

## Available MCP Servers

### Official Anthropic MCPs

Install with npm:

```bash
# Filesystem operations
npx -y @modelcontextprotocol/server-filesystem /path/to/dir

# PostgreSQL database
npx -y @modelcontextprotocol/server-postgres

# GitHub API
npx -y @modelcontextprotocol/server-github

# Git operations
npx -y @modelcontextprotocol/server-git

# Google Drive
npx -y @modelcontextprotocol/server-gdrive

# Slack
npx -y @modelcontextprotocol/server-slack
```

### Configuration Examples

#### PostgreSQL

```yaml
mcps:
  - name: postgres
    command: npx
    args: ["-y", "@modelcontextprotocol/server-postgres"]
    env:
      DATABASE_URL: "${DATABASE_URL}"
```

Tools provided:
- `postgres/query` - Execute SELECT queries
- `postgres/execute` - Execute INSERT/UPDATE/DELETE
- `postgres/list_tables` - List all tables
- `postgres/describe_table` - Describe table schema

#### GitHub

```yaml
mcps:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

Tools provided:
- `github/create_issue`
- `github/create_pull_request`
- `github/list_issues`
- `github/get_file_contents`

#### Filesystem

```yaml
mcps:
  - name: fs
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
```

Tools provided:
- `fs/read_file`
- `fs/write_file`
- `fs/list_directory`
- `fs/create_directory`

## Creating Custom MCPs

### Python Example

```python
# my_custom_mcp.py
from mcp.server import Server
from mcp.types import Tool, TextContent

server = Server("my-custom-mcp")

@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="send_slack_message",
            description="Send a message to Slack",
            inputSchema={
                "type": "object",
                "properties": {
                    "channel": {"type": "string"},
                    "message": {"type": "string"}
                },
                "required": ["channel", "message"]
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "send_slack_message":
        # Your logic here
        return [TextContent(type="text", text="Message sent!")]

if __name__ == "__main__":
    server.run()
```

### Usage in config.yaml

```yaml
mcps:
  - name: slack
    command: python
    args: ["./my_custom_mcp.py"]
    env:
      SLACK_TOKEN: "${SLACK_TOKEN}"
```

### Node.js Example

```javascript
// my-custom-mcps.js
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";

const server = new Server({
  name: "my-custom-mcps",
  version: "1.0.0"
}, {
  capabilities: {
    tools: {}
  }
});

server.setRequestHandler("tools/list", async () => {
  return {
    tools: [
      {
        name: "my_tool",
        description: "Does something useful",
        inputSchema: {
          type: "object",
          properties: {
            param: { type: "string" }
          },
          required: ["param"]
        }
      }
    ]
  };
});

server.setRequestHandler("tools/call", async (request) => {
  const { name, arguments: args } = request.params;

  if (name === "my_tool") {
    // Your logic here
    return {
      content: [
        { type: "text", text: "Result" }
      ]
    };
  }
});

const transport = new StdioServerTransport();
await server.connect(transport);
```

## Environment Variables

Use `${VAR_NAME}` syntax in config.yaml to reference environment variables:

```yaml
mcps:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"  # Read from environment
      GITHUB_ORG: "my-org"              # Static value
```

## Debugging

### Enable MCP Logs

MCP stderr output is automatically logged with prefix `🔧 MCP '<name>':`

### Check Active MCPs

```go
log.Printf("Active MCPs: %v", mcpRegistry.List())
```

### Verify Tools Registration

```go
log.Printf("Total tools: %d", len(tools.AllRegisteredTools()))
```

## Troubleshooting

### MCP Server Not Starting

1. Check command is in PATH
2. Verify arguments are correct
3. Check environment variables are set
4. Look for errors in logs

### Tools Not Available

1. Check MCP initialized successfully
2. Verify tools are registered: `tools.AllRegisteredTools()`
3. Check worker has access to toolkit

### Communication Errors

1. Ensure MCP implements JSON-RPC 2.0 correctly
2. Check stdio transport is working
3. Enable verbose logging

## Best Practices

1. **Use Global MCPs** for commonly needed tools (filesystem, web scraping)
2. **Use Worker MCPs** for specialized tools (database, APIs)
3. **Set Environment Variables** via `.env` file, not in config
4. **Test MCPs Independently** before integrating
5. **Handle Errors Gracefully** in custom MCPs
6. **Use Timeouts** for long-running operations
7. **Log Everything** for debugging

## Security Considerations

- MCPs run as separate processes with their own permissions
- Filesystem access can be restricted via MCP configuration
- Environment variables should be protected (use secrets management)
- Validate all inputs in custom MCPs
- Audit MCP tool calls like any other tool

## Performance

- MCPs communicate via stdin/stdout (fast)
- Each MCP is a separate process (isolation)
- Tools are cached after discovery (no repeated calls)
- Consider using connection pooling for database MCPs

## Roadmap

- [ ] SSE transport support (for remote MCPs)
- [ ] MCP marketplace/registry
- [ ] Hot reload of MCPs
- [ ] MCP health checks
- [ ] Rate limiting per MCP
- [ ] MCP sandboxing (resource limits)

## Resources

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Official MCP Servers](https://github.com/modelcontextprotocol/servers)
- [MCP Python SDK](https://github.com/modelcontextprotocol/python-sdk)
- [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)
