package handlers

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/speedtest002/azunyan-2-go/service"
)

func AutoQrHandler(qr *service.VietQRService) bot.EventListener {
	cleanRe := regexp.MustCompile(`(?i)<(?:@[!&]?|#|a?:\w+:|t:)\d+(?::[a-zA-Z])?>|https?://\S+`)
	stkRe := regexp.MustCompile(`\b\d{6,19}\b`)

	return bot.NewListenerFunc(func(e *events.MessageCreate) {
		if e.Message.Author.Bot {
			return
		}

		msg := strings.ToLower(cleanRe.ReplaceAllString(e.Message.Content, " "))
		if strings.TrimSpace(msg) == "" {
			return
		}

		stk := stkRe.FindString(msg)
		if stk == "" {
			return
		}

		words := strings.Fields(msg)
		var bank *service.Bank
		for _, w := range words {
			w = strings.Trim(w, ".,:;!?()[]{}\"'")
			if b, ok := qr.BankMap[w]; ok {
				bank = b
				break
			}
		}
		if bank == nil {
			return
		}

		embed := service.BuildQREmbed(bank.Bin, stk, 0, "", e.Message.Author)
		if _, err := e.Client().Rest.CreateMessage(e.ChannelID, discord.NewMessageCreate().AddEmbeds(embed)); err != nil {
			slog.Error("auto_qr: failed to send message", "channel_id", e.ChannelID, "guild_id", e.GuildID, "user_id", e.Message.Author.ID, "err", err)
		}
	})
}
