package clients

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/runtime"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
	"GoWorkerAI/app/workers"
)

var _ Interface = &DiscordClient{}

type DiscordClient struct {
	Client
	session   *discordgo.Session
	channelID string
}

func NewDiscordClient() *DiscordClient {
	token := os.Getenv("DISCORD_TOKEN")

	if token == "" {
		log.Fatal("DISCORD_TOKEN is not set")
	}

	session, _ := discordgo.New("Bot " + token)
	dc := &DiscordClient{
		session:   session,
		channelID: os.Getenv("DISCORD_CHANNEL_ID"),
	}

	session.AddHandler(dc.onMessageCreate)
	session.AddHandler(dc.onInteractionCreate)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	return dc
}

func (c *DiscordClient) Subscribe(rt *runtime.Runtime) {
	c.runtime = rt
	discordActions := []tools.Tool{
		{
			Name:        "send_discord_message",
			Description: "Use this action to send a text message to a specific Discord channel.",
			Parameters: tools.Parameter{
				Type: "object",
				Properties: map[string]any{
					"channel_id": map[string]any{
						"type":        "string",
						"description": fmt.Sprintf("Discord channel ID where the message will be sent. Default channel is %s", c.channelID),
					},
					"message": map[string]any{
						"type":        "string",
						"description": "The content of the message to send.",
					},
				},
				Required: []string{"channel_id", "message"},
			},
			HandlerFunc: func(tool tools.ToolTask) (string, error) {
				discordParams, err := utils.CastAny[discordParameters](tool.Parameters)
				if err != nil {
					return "", err
				}

				err = c.SendMessage(discordParams.ChannelID, discordParams.Message)

				return "✅ Message successfully sent to Discord channel " + discordParams.ChannelID, nil
			},
		},
	}

	c.runtime.AddTools(discordActions)
	c.Open()
}

type discordParameters struct {
	Message   string `json:"message"`
	ChannelID string `json:"channel_id"`
}

func (c *DiscordClient) Open() error {
	if err := c.session.Open(); err != nil {
		return err
	}
	log.Println("Discord client started. Listening for messages and interactions...")
	return nil
}

func (c *DiscordClient) Close() error {
	return c.session.Close()
}

func (c *DiscordClient) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()
	if m.Author.ID == s.State.User.ID {
		return
	}
	content := fmt.Sprintf("ChannelID: %s\nUserID %s\nUserName: %s\nMessage: %s",
		m.ChannelID, m.Author.ID, m.Author.Username, m.Content)
	if err := c.runtime.SaveEventOnHistory(ctx, content, models.UserRole); err != nil {
		log.Printf("⚠️ Error saving event: %v", err)
	}
	contentSplitted := strings.Fields(m.Content)
	var msg string
	switch strings.ToLower(contentSplitted[0]) {
	case "status":
		msg = c.getStatus(ctx, s, m)
	case "help", "!help":
		msg = "Supported commands: !help, !task"
	case "!task":
		if len(contentSplitted) < 2 {
			msg = "Usage: !task create <description> | !task cancel | !task status"
			break
		}
		if m.Author.ID != os.Getenv("DISCORD_ADMIN") {
			msg = "You are not authorized to use this command."
			break
		}

		cmd := contentSplitted[1]
		switch cmd {
		case "create":
			description := strings.Join(contentSplitted[2:], " ")
			description = description + "\n Rule: Must notify on discord (channel id: " + m.ChannelID + ") when you finish."
			newTask := workers.Task{
				Task:          description,
				MaxIterations: 5,
			}
			ev := runtime.Event{
				Task:        &newTask,
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.NewTask],
			}
			c.runtime.QueueEvent(ev)
			msg = "New task created, processing..."
		case "cancel":
			ev := runtime.Event{
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.CancelTask],
			}
			c.runtime.QueueEvent(ev)
			msg = "Active task cancelled."
		case "status":
			msg = c.getStatus(ctx, s, m)
		default:
			msg = "Unknown task command. Use: !task with create | cancel | status"
		}
	default:
		isMentioned := false
		for _, mention := range m.Mentions {
			if mention.ID == s.State.User.ID {
				isMentioned = true
			}
		}
		if !isMentioned {
			return
		}
		msg = c.runtime.ProcessQuickEvent(ctx, content)
	}
	s.ChannelMessageSend(m.ChannelID, msg)
}

func (c *DiscordClient) getStatus(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) string {
	s.ChannelMessageSend(m.ChannelID, "Processing...")
	return c.runtime.GetTaskStatus(ctx)
}

func (c *DiscordClient) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "task":
			c.handleTaskCommand(s, i)
		default:
			log.Printf("Unhandled command: %s", i.ApplicationCommandData().Name)
		}
	}
}

func (c *DiscordClient) handleTaskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		c.respondInteraction(s, i, "Usage: /task create <text> | /task cancel")
		return
	}
}

func (c *DiscordClient) respondInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	if err != nil {
		log.Printf("Error responding to slash command: %v", err)
	}
}

func (c *DiscordClient) SendMessage(channelID, content string) error {
	if channelID == "" {
		return fmt.Errorf("channelID is empty")
	}
	if _, err := c.session.ChannelMessageSend(channelID, content); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}
