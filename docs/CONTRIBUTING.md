# Contributing to GoWorkerAI

Thank you for your interest in contributing! This guide will help you get started.

---

## Ways to Contribute

- 🐛 **Report bugs** - Open an issue with details
- ✨ **Request features** - Share your ideas
- 📚 **Improve docs** - Fix typos, add examples
- 🔧 **Add native tools** - Extend built-in capabilities
- 🔌 **Create MCPs** - Build plugins for the community
- 🧪 **Write tests** - Improve code coverage
- 🎨 **Enhance UX** - Better logging, error messages

---

## Getting Started

### 1. Fork & Clone

```bash
# Fork on GitHub, then:
git clone https://github.com/YOUR_USERNAME/GoWorkerAI.git
cd GoWorkerAI
go mod tidy
```

### 2. Create a Branch

```bash
git checkout -b feature/amazing-feature
# or
git checkout -b fix/bug-description
```

### 3. Make Changes

See sections below for specific contribution types.

### 4. Test

```bash
# Run the app
go run .

# Build to check compilation
go build .

# Run tests (if available)
go test ./...
```

### 5. Commit

```bash
git add .
git commit -m "feat: add amazing feature"
# or
git commit -m "fix: resolve issue with X"
```

**Commit message format:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

### 6. Push & PR

```bash
git push origin feature/amazing-feature
```

Then open a Pull Request on GitHub with:
- Clear title describing the change
- Description of what and why
- Screenshots/examples if applicable
- Link to related issues

---

## Adding Native Tools

Native tools are built-in Go functions available to workers.

### 1. Create Tool File

```go
// app/tools/my_tool.go
package tools

func init() {
    Register(Tool{
        Name: "my_awesome_tool",
        Description: "Does something useful",
        Parameters: Parameter{
            Type: "object",
            Properties: map[string]any{
                "input": map[string]any{
                    "type":        "string",
                    "description": "Input parameter",
                },
            },
            Required: []string{"input"},
        },
        HandlerFunc: handleMyTool,
    })
}

func handleMyTool(task ToolTask) (string, error) {
    // Parse parameters
    input, ok := task.Parameters["input"].(string)
    if !ok {
        return "", fmt.Errorf("input must be a string")
    }

    // Do something useful
    result := process(input)

    // Return result
    return result, nil
}
```

### 2. Add to Preset (Optional)

```go
// app/tools/tools.go
case PresetMyTools:
    return pick(
        "my_awesome_tool",
        "another_tool",
    )
```

### 3. Test

```yaml
# config.yaml
members:
  - key: tester
    tools_preset: custom
    # Tool will be auto-registered
```

---

## Creating MCP Plugins

MCPs are external processes that provide tools. See [MCP Guide](MCP_GUIDE.md) for details.

### Quick Example (Python)

```python
# my_mcp.py
from mcp.server import Server
from mcp.types import Tool, TextContent

server = Server("my-mcp")

@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="hello",
            description="Say hello",
            inputSchema={
                "type": "object",
                "properties": {
                    "name": {"type": "string"}
                },
                "required": ["name"]
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "hello":
        return [TextContent(
            type="text",
            text=f"Hello, {arguments['name']}!"
        )]

if __name__ == "__main__":
    server.run()
```

---

## Adding New Worker Types

Workers are specialized agents that perform specific tasks.

### 1. Create Worker File

```go
// app/teams/workers/my_worker.go
package workers

type MyWorker struct {
    Base
}

func (w *MyWorker) GetSystemPrompt(context string) string {
    sys := `You are a specialized worker that does X.

Your responsibilities:
- Task 1
- Task 2

Guidelines:
- Rule 1
- Rule 2
`

    // Add rules if configured
    if w.Rules != nil {
        sys += "\n\nADDITIONAL RULES:\n"
        sys += strings.Join(w.Rules, "\n")
    }

    // Add context if provided
    if context != "" {
        sys += "\n\nCONTEXT:\n" + context
    }

    return sys
}
```

### 2. Register in Config Loader

```go
// app/config/loader.go
case "my_worker":
    return &workers.MyWorker{Base: base}, nil
```

### 3. Use in Config

```yaml
members:
  - key: specialist
    worker_type: my_worker
    when_call: "When specialized work is needed"
    tools_preset: file_basic
```

---

## Documentation Standards

### Code Comments

```go
// FunctionName does something useful.
// It takes X and returns Y.
// Example usage:
//   result, err := FunctionName(input)
func FunctionName(input string) (string, error) {
    // Implementation
}
```

### README Updates

- Keep language simple and clear
- Use examples for complex concepts
- Update table of contents
- Test all code snippets

### Config Examples

- Comment all non-obvious settings
- Provide multiple use cases
- Include environment variable references

---

## Testing Guidelines

### Manual Testing

1. Test with the default config
2. Test with custom configs
3. Test edge cases (empty inputs, large files, etc.)
4. Test error handling

### Automated Tests (TODO)

```go
// tools/my_tool_test.go
func TestMyTool(t *testing.T) {
    task := ToolTask{
        Key: "my_awesome_tool",
        Parameters: map[string]any{
            "input": "test",
        },
    }

    result, err := handleMyTool(task)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result != "expected" {
        t.Errorf("got %s, want expected", result)
    }
}
```

---

## Code Style

### Go Code

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions small and focused
- Handle errors explicitly
- Add comments for exported functions

### YAML Config

- Use 2 spaces for indentation
- Comment complex sections
- Group related settings
- Use meaningful names

### Naming Conventions

- **Workers**: `MyWorker` (PascalCase)
- **Tools**: `my_tool` (snake_case)
- **Config keys**: `worker_type` (snake_case)
- **Functions**: `GetSystemPrompt` (PascalCase)

---

## Pull Request Checklist

Before submitting:

- [ ] Code compiles without errors
- [ ] Tested manually with different scenarios
- [ ] Updated relevant documentation
- [ ] Added code comments
- [ ] Followed code style guidelines
- [ ] Commit messages are clear
- [ ] No sensitive data in commits

---

## Review Process

1. **Automated checks** - Build verification
2. **Code review** - Maintainer feedback
3. **Discussion** - Clarifications if needed
4. **Approval** - Merged when ready

**Response time:** We aim to review within 1-3 days.

---

## Community Guidelines

- Be respectful and constructive
- Help others in issues and discussions
- Share your use cases and examples
- Report bugs with detailed info
- Suggest features with context

---

## Questions?

- 💬 Open a discussion on GitHub
- 🐛 Report bugs in issues
- 📧 Email maintainers (see README)

---

**Thank you for contributing to GoWorkerAI!** 🎉
