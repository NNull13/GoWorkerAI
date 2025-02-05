package clients

import (
	"fmt"
	"log"
	"os"
	"strings"

	"GoWorkerAI/app/runtime"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
	"GoWorkerAI/app/workers"

	"github.com/bwmarrin/discordgo"
)

type DiscordClient struct {
	session *discordgo.Session
	runtime *runtime.Runtime
}

func NewDiscordClient() *DiscordClient {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil
	}

	session, _ := discordgo.New("Bot " + token)
	dc := &DiscordClient{
		session: session,
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
						"description": "Discord channel ID where the message will be sent. Use Default 1324515336980004949",
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
				if err != nil {
					return "", err
				}
				return "Message successfully sent to Discord channel " + discordParams.ChannelID, nil
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
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Author.ID != os.Getenv("DISCORD_ADMIN") {
		return
	}
	if strings.HasPrefix(m.Content, "!task") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: !task create <description> | !task cancel")
			return
		}

		cmd := parts[1]

		switch cmd {
		case "create":
			description := strings.Join(parts[2:], " ")
			newTask := workers.Task{
				Task:          description,
				MaxIterations: 5,
			}
			ev := runtime.Event{
				Task:        &newTask,
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.NewTask],
			}
			c.runtime.QueueEvent(ev)
			s.ChannelMessageSend(m.ChannelID, "New task created: "+description)

		case "cancel":
			ev := runtime.Event{
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.CancelTask],
			}
			c.runtime.QueueEvent(ev)
			s.ChannelMessageSend(m.ChannelID, "Active task cancelled.")

		default:
			s.ChannelMessageSend(m.ChannelID, "Unknown task command. Use: create | cancel")
		}
	}
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

// respondInteraction sends a response back to the user for a slash command interaction.
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
