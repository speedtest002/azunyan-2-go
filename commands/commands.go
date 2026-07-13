package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/speedtest002/azunyan-2-go/azunyan2go"
)

var Commands = []discord.ApplicationCommandCreate{
	versionCommand,
	pingCommand,
	avatarCommand,
	avatarUserCommand,
	sleepCommand,
	whatanimeSlashCommand,
	whatanimeMessageCommand,
	qrCommand,
}

func Register(h handler.Router, b *azunyan2go.Bot) {
	h.Command("/version", VersionHandler(b))

	h.Command("/ping", PingHandler)

	h.SlashCommand("/avatar", AvatarHandler)
	h.UserCommand("/avatar", AvatarUserHandler)

	h.SlashCommand("/sleep", SleepHandler)

	h.Command("/whatanime", WhatanimeHandler(b))

	h.Route("/qr", func(cr handler.Router) {
		cr.SlashCommand("/save", qrSaveHandler(b))
		cr.SlashCommand("/send", qrSendHandler(b))
	})
}
