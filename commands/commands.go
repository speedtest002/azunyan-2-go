package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/speedtest002/azunyan-2-go/azunyan2go"
)

var Commands = []discord.ApplicationCommandCreate{
	versionCommand,
	pingCommand,
}

func Register(h handler.Router, b *azunyan2go.Bot) {
	h.Command("/version", VersionHandler(b))

	h.Command("/ping", PingHandler)
}
