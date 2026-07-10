package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var avatarCommand = discord.SlashCommandCreate{
	Name:        "avatar",
	Description: "Get user's avatar",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionUser{
			Name:        "user",
			Description: "get user's avatar",
			Required:    false,
		},
	},
}

var avatarUserCommand = discord.UserCommandCreate{
	Name: "avatar",
}

func AvatarHandler(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	var member discord.Member
	m, ok := data.OptMember("user")
	if !ok {
		if e.Member() != nil {
			member = e.Member().Member
		} else {
			member = discord.Member{User: e.User()}
		}
	} else {
		member = m.Member
	}
	return createAvatarMessage(e, member)
}

func AvatarUserHandler(data discord.UserCommandInteractionData, e *handler.CommandEvent) error {
	member := data.TargetMember().Member
	if member.User.ID == 0 {
		member = discord.Member{User: data.TargetUser()}
	}

	return createAvatarMessage(e, member)
}

func createAvatarMessage(e *handler.CommandEvent, member discord.Member) error {
	embed := discord.NewEmbed().
		WithTitle(member.EffectiveName()).
		WithColor(0x00FFFF).
		WithImage(member.EffectiveAvatarURL(discord.WithSize(1024)))

	return e.CreateMessage(discord.NewMessageCreate().AddEmbeds(embed))
}
