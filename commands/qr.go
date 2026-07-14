package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/json"
	"github.com/speedtest002/azunyan-2-go/azunyan2go"
)

var qrCommand = discord.SlashCommandCreate{
	Name:        "qr",
	Description: "QR code commands",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "send",
			Description: "send VietQR",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:        "amount",
					Description: "số tiền",
					MaxValue:    json.Ptr(13),
					Required:    false,
				},
				discord.ApplicationCommandOptionString{
					Name:        "decription",
					Description: "nội dung chuyển khoản",
					MaxLength:   json.Ptr(50),
					Required:    false,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "save",
			Description: "save bank information",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "account_number",
					Description: "số tài khoản",
					MaxLength:   json.Ptr(19),
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "bank",
					Description: "ngân hàng",
					Required:    true,
				},
			},
		},
	},
}

func qrSaveHandler(b *azunyan2go.Bot) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		userId := int64(e.User().ID)
		accountNumber, _ := data.OptString("account_number")
		bank, _ := data.OptString("bank")

		err := b.DB.UpsertUserBankInfo(e.Ctx, userId, accountNumber, bank)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("Error: %s", err)}.WithEphemeral(true))
		}

		return e.CreateMessage(discord.MessageCreate{Content: "Saved!"}.WithEphemeral(true))
	}
}

func qrSendHandler(b *azunyan2go.Bot) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		userId := int64(e.User().ID)
		userBankInfo, err := b.DB.GetUserBankInfo(e.Ctx, userId)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("Error: %s", err)}.WithEphemeral(true))
		}

		params := url.Values{}

		if amount, ok := data.OptInt("ammount"); ok {
			params.Set("amount", strconv.Itoa(amount))
		}

		if description, ok := data.OptString("description"); ok {
			params.Set("addInfo", description)
		}

		reqURL := fmt.Sprintf(
			"https://img.vietqr.io/image/%s-%s-print.jpg?%s",
			userBankInfo.Bank,
			userBankInfo.AccountNumber,
			params.Encode(),
		)

		embed := discord.NewEmbed().
			WithImage(reqURL).
			WithColor(0x00FFFF).
			WithDescription("Mã QR của <@" + e.User().ID.String() + ">")

		return e.CreateMessage(discord.NewMessageCreate().AddEmbeds(embed))
	}
}
