package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var versionCommand = discord.SlashCommandCreate{
	Name:        "version",
	Description: "version command",
}

func VersionHandler(version, commit string) handler.CommandHandler {
	return func(e *handler.CommandEvent) error {
		return e.CreateMessage(discord.MessageCreate{
			Content: fmt.Sprintf("Version: %s\nCommit: %s", version, commit),
		})
	}
}
