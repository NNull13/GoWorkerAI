# GoWorkerAI 🤖

> A flexible, extensible AI agent framework for autonomous task execution with specialized workers and MCP plugin support.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## ✨ What is GoWorkerAI?

GoWorkerAI is a **multi-agent orchestration framework** where specialized AI workers collaborate to complete complex tasks autonomously. Built in Go for performance, with YAML configuration for flexibility.

### Key Features

- 🔥 **Multi-Agent System** - Leader coordinates specialized workers (Coder, FileManager, etc.)
- 🔌 **MCP Protocol** - Extend with plugins in any language (Python, Node.js, Rust)
- 🛠️ **Native Tools** - Built-in file operations, web scraping, and more
- 📝 **YAML Config** - Define teams and MCPs declaratively
- 🤝 **Local Model Optimized** - Works great with LM Studio, Ollama, etc.
- 💾 **Full History** - SQLite tracking with audit logs
- 🎯 **Discord Integration** - Optional bot for remote control

---

## 🚀 Quick Start

### 1. Install

```bash
git clone https://github.com/NNull13/GoWorkerAI.git
cd GoWorkerAI
go mod tidy
```

### 2. Configure

```bash
# Copy example config
cp config.example.yaml config.yaml

# Set your LLM endpoint
export LLM_BASE_URL="http://localhost:1234"
export WORKER_FOLDER="./playground"
```

### 3. Run

```bash
    node run-mcps.js

    go run .
```

**That's it!** The default team will start executing the configured task.

---

## 📖 Documentation

- **[Quick Start Guide](docs/QUICKSTART.md)** - Detailed tutorial with examples
- **[MCP Guide](docs/MCP_GUIDE.md)** - Create and use custom plugins
- **[Configuration Reference](config.example.yaml)** - All available options
- **[Contributing Guide](docs/CONTRIBUTING.md)** - How to contribute

---

## 🔌 MCP Plugin System

Extend functionality without recompiling:

```yaml
teams:
  default:
    members:
      - key: coder
        mcps:
          # Add PostgreSQL access
          - name: postgres
            command: npx
            args: ["-y", "@modelcontextprotocol/server-postgres"]
            env:
              DATABASE_URL: "${DATABASE_URL}"
```

See [MCP Guide](docs/MCP_GUIDE.md) for official plugins and how to create custom ones.

---

## 🛠️ Architecture

```
┌─────────────┐
│   Leader    │  Plans & delegates
└──────┬──────┘
       │
   ┌───┴────┬──────────┐
   ▼        ▼          ▼
┌──────┐ ┌──────┐ ┌──────┐
│Worker│ │Worker│ │Worker│
│      │ │      │ │      │
└──────┘ └──────┘ └──────┘
   │       │          │
   └───────┴──────────┘
           │
   ┌───────▼────────┐
   │  Native Tools  │
   │  + MCP Plugins │
   └────────────────┘
```

- **Leader**: Strategic planning and task delegation
- **Specialized Workers**: Execute specific types of work
- **Tools & MCPs**: Extend capabilities on demand

---

## 💡 Example Use Cases

- **Code Generation**: Automated project scaffolding
- **Data Analysis**: Query databases, process files
- **Web Automation**: Scrape data, interact with APIs
- **DevOps**: Deploy, monitor, manage infrastructure
- **Research**: Gather info, summarize, analyze

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

**Ways to contribute:**
- 🔧 Add native tools
- 🔌 Create MCP and/or plugins
- 📚 Improve documentation
- 🐛 Report bugs
- ✨ Request features

---

## 📝 License

This project is open source under the MIT License.

---

## 🙏 Acknowledgments

- Built with [Model Context Protocol](https://modelcontextprotocol.io/)
- Inspired by multi-agent systems and autonomous AI
- Thanks to all contributors!

---

### Crafted with ❤️ by [NoName13](https://github.com/NNull13)

**Questions?** [Open an issue](https://github.com/NNull13/GoWorkerAI/issues) • **Want updates?** Star the repo ⭐
