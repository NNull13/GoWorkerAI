# Client Connectors Guide

Clients are external connectors that allow GoWorkerAI to receive interactions and events from various platforms like Discord, Slack, Telegram, etc.

---

## Configuration

Clients are configured in `config.yaml` under the `clients` section:

```yaml
clients:
  - type: discord
    enabled: true
    config:
      token: "${DISCORD_TOKEN}"
      channel_id: "${DISCORD_CHANNEL_ID}"
      admin_id: "${DISCORD_ADMIN}"
```

---

## Available Clients

### Discord Bot

Connect GoWorkerAI to Discord for remote control and monitoring.

#### Setup

1. **Create a Discord Bot:**
   - Go to https://discord.com/developers/applications
   - Click "New Application"
   - Go to "Bot" tab and click "Add Bot"
   - Copy the token

2. **Configure Permissions:**
   Required bot permissions:
   - Send Messages
   - Read Message History
   - Add Reactions

3. **Invite to Server:**
   - Go to "OAuth2" > "URL Generator"
   - Select scopes: `bot`, `applications.commands`
   - Select permissions mentioned above
   - Copy and visit the generated URL

4. **Configure GoWorkerAI:**

```yaml
clients:
  - type: discord
    enabled: true
    config:
      token: "YOUR_BOT_TOKEN_HERE"
      channel_id: "CHANNEL_ID"  # Optional: Default channel for notifications
      admin_id: "YOUR_USER_ID"  # Optional: Admin user who can use !task commands
```

Or use environment variables:

```bash
export DISCORD_TOKEN="your_bot_token"
export DISCORD_CHANNEL_ID="channel_id"
export DISCORD_ADMIN="your_user_id"
```

Then:

```yaml
clients:
  - type: discord
    enabled: true
    config:
      token: "${DISCORD_TOKEN}"
      channel_id: "${DISCORD_CHANNEL_ID}"
      admin_id: "${DISCORD_ADMIN}"
```

#### Commands

**Public Commands:**
- `@BotName <message>` - Quick interaction without creating a task
- `status` - Get current task status
- `help` or `!help` - Show available commands

**Admin Commands:**
- `!task create <description>` - Create a new task
- `!task cancel` - Cancel the active task
- `!task status` - Get detailed task status

#### Example Usage

```
User: @GoWorkerAI what's the weather today?
Bot: I can help you with that, but I don't have access to weather APIs...

Admin: !task create Build a weather API client that fetches data from OpenWeather
Bot: New task created, processing...

User: status
Bot: Active task: Build a weather API client...
     Progress: Coder is implementing the HTTP client...
```

---

## Adding New Client Types

You can easily add new client connectors (Slack, Telegram, HTTP webhooks, etc.).

### 1. Implement Client Interface

Create a new file `app/clients/myclient.go`:

```go
package clients

import (
	"GoWorkerAI/app/runtime"
)

type MyClient struct {
	Client
	// Your fields here
}

func NewMyClientFromConfig(config map[string]string) (*MyClient, error) {
	// Initialize from config
	apiKey := config["api_key"]

	client := &MyClient{
		// Setup
	}

	return client, nil
}

func (c *MyClient) Subscribe(rt *runtime.Runtime) {
	c.runtime = rt
	// Setup event handlers
}

func (c *MyClient) Close() error {
	// Cleanup
	return nil
}
```

### 2. Register in Factory

Update `app/clients/registry.go`:

```go
func CreateClient(cfg Config) (Interface, error) {
	switch cfg.Type {
	case "discord":
		return NewDiscordClientFromConfig(cfg.Config)
	case "myclient":  // Add your client here
		return NewMyClientFromConfig(cfg.Config)
	default:
		return nil, fmt.Errorf("unknown client type: %s", cfg.Type)
	}
}
```

### 3. Add to Configuration

```yaml
clients:
  - type: myclient
    enabled: true
    config:
      api_key: "${MY_CLIENT_API_KEY}"
      webhook_url: "https://example.com/webhook"
```

That's it! Your new client will be automatically initialized.

---

## Client Interface

All clients must implement:

```go
type Interface interface {
	Subscribe(*runtime.Runtime)
}
```

Optional interfaces:
- `Close() error` - For cleanup

---

## Best Practices

### Security

- **Never hardcode tokens** in config files
- Use environment variables: `${DISCORD_TOKEN}`
- Add sensitive configs to `.gitignore`
- Rotate tokens regularly

### Error Handling

- Log errors but don't crash the app
- Implement reconnection logic
- Validate messages before processing

### Performance

- Process events asynchronously
- Don't block on I/O operations
- Use goroutines for concurrent handling

---

## Examples

### Discord with Quick Replies

```yaml
clients:
  - type: discord
    enabled: true
    config:
      token: "${DISCORD_TOKEN}"
```

Quick replies let users interact without creating full tasks:

```
User: @Bot explain what you can do
Bot: [Processes with EventHandler, responds immediately]
```

### Multiple Clients

```yaml
clients:
  - type: discord
    enabled: true
    config:
      token: "${DISCORD_TOKEN}"

  - type: slack
    enabled: true
    config:
      token: "${SLACK_TOKEN}"
```

Both will receive events and can trigger tasks.

---

## Troubleshooting

### Client Not Starting

**Check logs:**
```
❌ Failed to initialize clients: ...
```

**Common issues:**
- Token not set or invalid
- Missing permissions
- Network connectivity

### Events Not Received

- Verify bot is in the channel
- Check intents are configured
- Ensure handlers are registered

### Commands Not Working

- Check admin_id is set correctly
- Verify permissions
- Look for error logs

---

**Need help?** [Open an issue](https://github.com/NNull13/GoWorkerAI/issues)
