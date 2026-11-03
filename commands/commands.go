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
	reminderCommand,
}

func Register(h handler.Router, b *azunyan2go.Bot) {
	h.Command("/version", VersionHandler(b.Version, b.Commit))

	h.Command("/ping", PingHandler)

	h.SlashCommand("/avatar", AvatarHandler)
	h.UserCommand("/avatar", AvatarUserHandler)

	h.SlashCommand("/sleep", SleepHandler)

	h.Command("/whatanime", WhatanimeHandler(b.Paginator))

	h.Route("/bank", func(cr handler.Router) {
		cr.SlashCommand("/save", qrSaveHandler(b.DB))
		cr.SlashCommand("/qr", qrSendHandler(b.DB))
		cr.ButtonComponent("/deeplink", deeplinkButtonHandler(b.DB))
	})

	h.Route("/reminder", func(cr handler.Router) {
		cr.SlashCommand("/create", reminderCreateHandler(b.DB, b.Scheduler))
		cr.SlashCommand("/cancel", reminderCancelHandler(b.DB))
		cr.SlashCommand("/list", reminderListHandler(b.DB))
		cr.ButtonComponent("/join/{id}", joinReminderButtonReminder(b.DB))
		cr.ButtonComponent("/leave/{id}", leaveReminderButtonHandler(b.DB))
	})
}
