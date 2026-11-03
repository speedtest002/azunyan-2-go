package commands

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/json"

	"github.com/speedtest002/azunyan-2-go/db"
	"github.com/speedtest002/azunyan-2-go/service"
)

var qrCommand = discord.SlashCommandCreate{
	Name:        "bank",
	Description: "bank commands",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "qr",
			Description: "send qr VietQR",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionUser{
					Name:        "user",
					Description: "lấy qr của người khác",
					Required:    false,
				},
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
		targetUser := e.User()
		if u, ok := data.OptUser("user"); ok {
			targetUser = u
		}
		userID := int64(targetUser.ID)

		userBankInfo, err := database.GetUserBankInfo(e.Ctx, userID)
		if err != nil {
			_ = e.CreateMessage(discord.MessageCreate{Content: "error from server, please try later"}.WithEphemeral(true))
			return err
		}
		if userBankInfo.AccountNumber == "" || userBankInfo.Bank == "" {
			return e.CreateMessage(discord.MessageCreate{Content: "you have not saved your bank info yet, use `/bank save`"}.WithEphemeral(true))
		}

		amount, _ := data.OptInt("amount")
		description, _ := data.OptString("description")

		embed := service.BuildQREmbed(userBankInfo.Bank, userBankInfo.AccountNumber, amount, description, targetUser)

		return e.CreateMessage(discord.NewMessageCreate().
			AddEmbeds(embed).
			AddActionRow(service.DeeplinkButton()),
		)
	}
}

func deeplinkButtonHandler(database *db.DB) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		userBankInfo, err := database.GetUserBankInfo(e.Ctx, int64(e.User().ID))
		if err != nil {
			_ = e.CreateMessage(discord.NewMessageCreate().WithContent("error from server, please try later").WithEphemeral(true))
			return err
		}
		if userBankInfo.Bank == "" {
			return e.CreateMessage(discord.NewMessageCreate().
				WithContent("you have not saved your bank info yet, use `/bank save`").
				WithEphemeral(true))
		}

		if len(e.Message.Embeds) == 0 || e.Message.Embeds[0].Author == nil || e.Message.Embeds[0].Author.URL == "" {
			return e.CreateMessage(discord.NewMessageCreate().
				WithContent("cannot find payment info in this message").
				WithEphemeral(true))
		}

		dlURL, err := service.WithDeepLinkApp(e.Message.Embeds[0].Author.URL, userBankInfo.Bank)
		if err != nil {
			_ = e.CreateMessage(discord.NewMessageCreate().WithContent("failed to build payment link").WithEphemeral(true))
			return err
		}

		bankLabel := strings.ToUpper(userBankInfo.Bank)
		return e.CreateMessage(discord.NewMessageCreate().
			WithContent(fmt.Sprintf("Mở ứng dụng **%s** trên điện thoại để chuyển tiền\nHiện tại chỉ hỗ trợ điền nhanh thông tin chuyển khoản ở các ứng dụng VietinBank iPay, BIDV SmartBanking, OCB OMNI, ACB ONE ", bankLabel)).
			AddActionRow(
				discord.NewLinkButton("Android", dlURL),
				discord.NewLinkButton("iOS", dlURL),
			).
			WithEphemeral(true))
	}
}
