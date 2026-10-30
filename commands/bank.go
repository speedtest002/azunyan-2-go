package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/json"

	"github.com/speedtest002/azunyan-2-go/db"
	"github.com/speedtest002/azunyan-2-go/service"
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
					MinValue:    json.Ptr(1000),
					Required:    false,
				},
				discord.ApplicationCommandOptionString{
					Name:        "description",
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

func qrSaveHandler(database *db.DB) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		userId := int64(e.User().ID)
		accountNumber, _ := data.OptString("account_number")
		bank, _ := data.OptString("bank")

		err := database.UpsertUserBankInfo(e.Ctx, userId, accountNumber, bank)
		if err != nil {
			_ = e.CreateMessage(discord.MessageCreate{Content: "error saving your bank info, please try again later"}.WithEphemeral(true))
			return err
		}

		return e.CreateMessage(discord.MessageCreate{Content: "Saved!"}.WithEphemeral(true))
	}
}

func qrSendHandler(database *db.DB) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		userID := int64(e.User().ID)
		userBankInfo, err := database.GetUserBankInfo(e.Ctx, userID)
		if err != nil {
			_ = e.CreateMessage(discord.MessageCreate{Content: "error... thử lại sau"}.WithEphemeral(true))
			return err
		}
		if userBankInfo.AccountNumber == "" || userBankInfo.Bank == "" {
			return e.CreateMessage(discord.MessageCreate{Content: "you have not saved your bank info yet, use `/qr save`"}.WithEphemeral(true))
		}

		amount, _ := data.OptInt("amount")
		description, _ := data.OptString("description")

		embed := service.BuildQREmbed(userBankInfo.Bank, userBankInfo.AccountNumber, amount, description, e.User())

		return e.CreateMessage(discord.NewMessageCreate().AddEmbeds(embed))
	}
}
