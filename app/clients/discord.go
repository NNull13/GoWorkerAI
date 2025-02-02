package clients

import (
	"fmt"
	"log"
	"os"
	"strings"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/runtime"
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
	discordActions := []actions.Action{
		{
			Key: "send_discord_message",
			HandlerFunc: func(action *models.ActionTask, folder string) (result string, err error) {
				channelID := action.Filename
				messageText := action.Content
				err = c.SendMessage(channelID, messageText)
				if err != nil {
					return "", err
				}
				return "Message successfully sent to Discord channel " + channelID, nil
			},
			Description: "Use this action to send a message to a Discord channel.\n" +
				"You must provide 'channel_id' in the filename field, and 'content' as the message text.\n\n" +
				"**Example**:\n```json\n{\n  \"action\": \"send_discord_message\",\n  \"filename\": \"123456789012345678\",\n  \"content\": \"Hello from the AI!\"\n}\n```\n\n" +
				"When to use:\n" +
				"- You want to post or reply in a specific channel.\n" +
				"- You have the correct 'channel_id' to target.\n",
		},
	}
	c.runtime.AddActions(discordActions)
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
				Type:        "new_task",
				Task:        &newTask,
				HandlerFunc: runtime.EventsHandlerFuncDefault[runtime.NewTask],
			}
			c.runtime.QueueEvent(ev)
			s.ChannelMessageSend(m.ChannelID, "New task created: "+description)

		case "cancel":
			ev := runtime.Event{
				Type:        "cancel_task",
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
