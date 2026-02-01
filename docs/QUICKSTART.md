# Quick Start Guide

Get GoWorkerAI running in 5 minutes.

---

## Prerequisites

- Go 1.24+ installed
- Local LLM server running (LM Studio, Ollama, etc.)
- Optional: Node.js (for MCP plugins)

---

## Step 1: Installation

```bash
git clone https://github.com/NNull13/GoWorkerAI.git
cd GoWorkerAI
go mod tidy
```

---

## Step 2: Start Your LLM Server

### Option A: LM Studio
1. Download from https://lmstudio.ai
2. Load a model (recommended: Qwen 2.5 or similar)
3. Start server (default: http://localhost:1234)

### Option B: Ollama
```bash
ollama serve
ollama run qwen2.5:latest
```

---

## Step 3: Configure GoWorkerAI

### Basic Setup

```bash
# Copy the example config
cp config.example.yaml config.yaml

# Set environment variables
export LLM_BASE_URL="http://localhost:1234"
export LLM_MODEL="qwen2.5"
export WORKER_FOLDER="./playground"
```

### Edit config.yaml

```yaml
teams:
  default:
    task: "Create a simple calculator API in Go"

    members:
      - key: leader
        worker_type: leader

      - key: event_handler
        worker_type: event_handler

      - key: coder
        worker_type: coder
        when_call: "When code needs to be written"
        tools_preset: file_basic
```

---

## Step 4: Run

```bash
go run .
```

You should see:
```
📋 Loading configuration from: ./config.yaml
✅ Registered 14 builtin tools
🏗️ Building team: default
✅ All systems started. Press Ctrl+C to exit...
```

---

## Understanding the Output

GoWorkerAI will:
1. **Leader plans** the task breakdown
2. **Delegates** to the Coder worker
3. **Coder executes** using file tools
4. **Reports back** to Leader
5. **Validates** completion

Check `./playground` for generated files!

---

## Next Steps

### Try Different Tasks

Edit `config.yaml` and change the task:

```yaml
task: "Create a REST API with user authentication"
# or
task: "Build a CLI tool that fetches weather data"
# or
task: "Write unit tests for existing code in ./myproject"
```

### Add More Workers

```yaml
members:
  - key: researcher
    worker_type: coder
    when_call: "When research or information gathering is needed"
    tools_preset: scraper_basic
```

### Use MCPs (Advanced)

Add external capabilities:

```yaml
members:
  - key: coder
    mcps:
      - name: github
        command: npx
        args: ["-y", "@modelcontextprotocol/server-github"]
        env:
          GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

See [MCP Guide](MCP_GUIDE.md) for details.

---

## Common Issues

### "Failed to load config"
**Fix:** Check that `config.yaml` exists and is valid YAML.
```bash
# Validate YAML syntax
cat config.yaml | python -c "import yaml, sys; yaml.safe_load(sys.stdin)"
```

### "Connection refused" to LLM
**Fix:** Ensure your LLM server is running.
```bash
curl http://localhost:1234/v1/models
```

### "No tools available"
**Fix:** Check logs for tool registration errors.

### Worker keeps failing
**Fix:**
- Check model has enough context window
- Simplify the task
- Add more specific rules to the worker

---

## Configuration Deep Dive

### Worker Types

| Type | Purpose | Default Tools |
|------|---------|---------------|
| `leader` | Plans & delegates | delegate |
| `coder` | Writes code | file_basic |
| `file_manager` | File operations | file_basic |
| `event_handler` | Handles events | none |

### Tool Presets

| Preset | Tools Included |
|--------|----------------|
| `delegate` | delegate_task |
| `minimal` | read_file, list_files |
| `readonly` | minimal + search_file |
| `file_basic` | read, write, delete, list, mkdir |
| `scraper_basic` | fetch_html, extract_text, extract_links |
| `all` | All available tools |

### Environment Variables

```bash
# LLM Configuration
export LLM_BASE_URL="http://localhost:1234"
export LLM_MODEL="qwen2.5"
export LLM_EMBEDDINGS_MODEL="nomic-embed-text"

# Worker Configuration
export WORKER_FOLDER="./playground"  # Sandbox directory

# Optional
export CONFIG_PATH="./my-config.yaml"
export TEAM_NAME="custom-team"
export DB_PATH="./data/records.db"
```

---

## Example Workflows

### 1. Code Generation

```yaml
task: "Create a REST API with the following endpoints: /users (GET, POST), /login (POST)"
```

### 2. Data Processing

```yaml
task: "Read all CSV files in ./data/, combine them, and generate a summary report"
```

### 3. Web Scraping

```yaml
task: "Scrape product prices from example.com and save to a JSON file"
members:
  - key: scraper
    worker_type: coder
    tools_preset: scraper_basic
```

### 4. Testing

```yaml
task: "Generate unit tests for all Go files in ./src/ directory"
```

---

## Performance Tips

1. **Use appropriate models**: Larger models (7B+) work better
2. **Clear tasks**: Be specific about what you want
3. **Constrain scope**: Break large tasks into smaller ones
4. **Monitor logs**: Check `./logs/` for debugging

---

## What's Next?

- 📖 [MCP Guide](MCP_GUIDE.md) - Add custom plugins
- 🔧 [Configuration Reference](../config.example.yaml) - All options
- 🤝 [Contributing](CONTRIBUTING.md) - Help improve GoWorkerAI
- 💬 [Discord Integration](DISCORD.md) - Remote control

---

**Need help?** [Open an issue](https://github.com/NNull13/GoWorkerAI/issues)
