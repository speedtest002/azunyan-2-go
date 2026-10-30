package commands

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/paginator"

	"github.com/speedtest002/azunyan-2-go/service"
)

var whatanimeSlashCommand = discord.SlashCommandCreate{
	Name:        "whatanime",
	Description: "find anime from screenshot",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionAttachment{
			Name:        "image",
			Description: "screenshoot",
			Required:    true,
		},
	},
}

var whatanimeMessageCommand = discord.MessageCommandCreate{
	Name: "whatanime",
}

func WhatanimeHandler(p *paginator.Manager) handler.CommandHandler {
	return func(e *handler.CommandEvent) error {
		var imageURL string
		switch data := e.Data.(type) {
		case discord.SlashCommandInteractionData:
			att, ok := data.OptAttachment("image")
			if !ok {
				return e.CreateMessage(discord.MessageCreate{Content: "No image?"})
			}
			imageURL = att.URL
		case discord.MessageCommandInteractionData:
			msg := data.TargetMessage()
			for _, att := range msg.Attachments {
				if att.ContentType != nil && strings.HasPrefix(*att.ContentType, "image/") {
					imageURL = att.URL
					break
				}
			}
			if imageURL == "" {
				return e.CreateMessage(discord.MessageCreate{Content: "No image found in message."})
			}
		}

		if err := e.DeferCreateMessage(false); err != nil {
			return err
		}

		body, err := service.FetchTraceMoe(imageURL)
		if err != nil {
			_, _ = e.CreateFollowupMessage(discord.MessageCreate{Content: "Failed to search anime on trace.moe, please try again later."})
			return err
		}

		traceResult, err := service.BuildTraceMoeAPI(body)
		if err != nil {
			_, _ = e.CreateFollowupMessage(discord.MessageCreate{Content: "Failed to parse response from trace.moe, please try again later."})
			return err
		}

		filtered := slices.DeleteFunc(traceResult.Result, func(r service.TraceMoeResult) bool {
			return r.Similarity < 0.7
		})
		if len(filtered) == 0 {
			_, err = e.CreateFollowupMessage(discord.MessageCreate{Content: "No results found."})
			return err
		}
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}

		return p.Create(func(_ discord.InteractionResponseType, data discord.InteractionResponseData, opts ...rest.RequestOpt) error {
			msg, ok := data.(discord.MessageCreate)
			if !ok {
				return nil
			}
			_, err := e.CreateFollowupMessage(msg, opts...)
			return err
		}, paginator.Pages{
			ID: e.ID().String(),
			PageFunc: func(page int, embed discord.Embed) discord.Embed {
				r := filtered[page]
				embed.Color = 0x2ECC71
				embed.Title = r.Anilist.Title.Native
				embed.Description = fmt.Sprintf(
					"# %s\n## %v\n`%s`\n%s\n[Preview video](%s)",
					r.Anilist.Title.Romaji,
					r.Anilist.Title.English,
					r.Filename,
					time.Duration(r.At*float64(time.Second)),
					r.Video,
				)
				embed.Image = &discord.EmbedResource{URL: r.Image}
				embed.Footer = &discord.EmbedFooter{
					Text: fmt.Sprintf("Page %d/%d | Powered by trace.moe", page+1, len(filtered)),
				}
				return embed
			},
			Pages:      len(filtered),
			Creator:    e.User().ID,
			ExpireMode: paginator.ExpireModeAfterLastUsage,
		}, false)
	}
}
