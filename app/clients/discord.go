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
	"GoWorkerAI/app/teams"
)

var _ Interface = &DiscordClient{}

const originDiscord string = "discord"

type DiscordClient struct {
	Client
	session   *discordgo.Session
	channelID string
}

func NewDiscordClientFromConfig(config map[string]string) (*DiscordClient, error) {
	token := config["token"]
	if token == "" {
		token = os.Getenv("DISCORD_TOKEN")
	}

	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN not provided in config or environment")
	}

	channelID := config["channel_id"]
	if channelID == "" {
		channelID = os.Getenv("DISCORD_CHANNEL_ID")
	}

	adminID := config["admin_id"]
	if adminID == "" {
		adminID = os.Getenv("DISCORD_ADMIN")
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	dc := &DiscordClient{
		session:   session,
		channelID: channelID,
	}

	session.AddHandler(dc.onMessageCreate)
	session.AddHandler(dc.onInteractionCreate)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	log.Printf("✅ Discord client configured (channel: %s)\n", channelID)
	return dc, nil
}

func (c *DiscordClient) Subscribe(rt *runtime.Runtime) {
	c.runtime = rt
	c.Open()
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
		msg = c.getStatus(s, m)
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
			newTask := teams.Task{
				Description: description,
			}
			ev := runtime.Event{
				Origin:      originDiscord,
				Task:        &newTask,
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.NewTask],
			}
			c.runtime.QueueEvent(ev)
			msg = "New task created, processing..."
		case "cancel":
			ev := runtime.Event{
				Origin:      originDiscord,
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.CancelTask],
			}
			c.runtime.QueueEvent(ev)
			msg = "Active task cancelled."
		case "status":
			msg = c.getStatus(s, m)
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

func (c *DiscordClient) getStatus(s *discordgo.Session, m *discordgo.MessageCreate) string {
	s.ChannelMessageSend(m.ChannelID, "Processing...")
	return c.runtime.GetTaskStatus()
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
