package handlers

import (
	"github.com/speedtest002/azunyan-2-go/azunyan2go"
)

func Register(b *azunyan2go.Bot) {
	b.Client.AddEventListeners(
		AutoQrHandler(b.VietQR),
	)
}