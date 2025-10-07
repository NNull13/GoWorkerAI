package tools

/*

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



send_discord_message: {
		Name:        send_discord_message,
		Description: "Use this action to send a text message to a specific Discord channel.",
		Parameters: Parameter{
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
		HandlerFunc: executeDiscordAction,
	},
*/
