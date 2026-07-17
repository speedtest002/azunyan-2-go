package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/speedtest002/azunyan-2-go/azunyan2go"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func AutoQrHandler(b *azunyan2go.Bot) bot.EventListener {
	re := regexp.MustCompile(`\b\d{6,19}\b`)

	return bot.NewListenerFunc(func(e *events.MessageCreate) {
		if e.Message.Author.Bot {
			return
		}

		msg := strings.ToLower(e.Message.Content)
		if msg == "" {
			return
		}

		stk := re.FindString(msg)
		if stk == "" {
			return
		}

		words := strings.Fields(msg)
		var bank *azunyan2go.Bank
		for _, w := range words {
			if b, ok := b.BankMap[w]; ok {
				bank = b
				break
			}
		}
		if bank == nil {
			return
		}

		url := fmt.Sprintf("https://img.vietqr.io/image/%s-%s-print.jpg", bank.Bin, stk)

		name := e.Message.Author.Username
		if e.Message.Author.GlobalName != nil {
			name = *e.Message.Author.GlobalName
		}
		embed := discord.NewEmbed().WithImage(url).WithColor(0x00FFFF).WithAuthor(name, "", "")
		e.Client().Rest.CreateMessage(e.ChannelID, discord.NewMessageCreate().AddEmbeds(embed))
	})
}

func FetchQrHandler(b *azunyan2go.Bot) bot.EventListener {
	return bot.NewListenerFunc(func(e *events.Ready) {
		banks, err := fetchVietQR()
		if err != nil {
			slog.Error("Không thể fetch dữ liệu từ VietQR", "error", err)
			return
		}
		b.Banks = banks
		b.BankMap = make(map[string]*azunyan2go.Bank, len(banks)*3)
		for i := range banks {
			bank := &banks[i]
			b.BankMap[strings.ToLower(bank.ShortName)] = bank
			b.BankMap[strings.ToLower(bank.Code)] = bank
		}
		slog.Info("loaded banks", slog.Int("count", len(banks)))
	})
}

const vietqrAPI = "https://api.vietqr.io/v2/banks"

type vietQRResponse struct {
	Data []azunyan2go.Bank `json:"data"`
}

func fetchVietQR() ([]azunyan2go.Bank, error) {
	resp, err := http.Get(vietqrAPI)

	if err != nil {
		return nil, fmt.Errorf("Có lỗi đã xảy ra: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Lỗi: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Có lỗi khi đọc body: %w", err)
	}

	var apiResp vietQRResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("Có lỗi khi parse json: %w", err)
	}
	return apiResp.Data, nil
}
