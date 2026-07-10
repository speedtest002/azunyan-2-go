package commands

import (
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var pingCommand = discord.SlashCommandCreate{
	Name:        "ping",
	Description: "Ping pong ping pong",
}

func PingHandler(e *handler.CommandEvent) error {
	return e.CreateMessage(
		discord.MessageCreate{
			Content: ping(e),
		},
	)
}

func ping(e *handler.CommandEvent) string {
	rest_start := time.Now()
	_, err := e.Client().Rest.GetGatewayBot()
	if err != nil {
		return "An unknow error when using this command"
	}
	rest_latency := time.Since(rest_start)
	return fmt.Sprintf("Pong! %v", rest_latency.String())
}
