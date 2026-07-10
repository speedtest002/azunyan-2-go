package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/paginator"

	"github.com/speedtest002/azunyan-2-go/azunyan2go"
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

func WhatanimeHandler(b *azunyan2go.Bot) handler.CommandHandler {
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

		body, err := fetchTraceMoe(imageURL)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: err.Error()})
		}

		traceResult, err := buildTraceMoeAPI(body)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: err.Error()})
		}

		filtered := slices.DeleteFunc(traceResult.Result, func(r TraceMoeResult) bool {
			return r.Similarity < 0.7
		})
		if len(filtered) == 0 {
			return e.CreateMessage(discord.MessageCreate{Content: "No results found."})
		}
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}

		return b.Paginator.Create(e.Respond, paginator.Pages{
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

const traceMoeBase = "https://api.trace.moe/search"

type TraceMoeResult struct {
	Anilist    AnilistInfo `json:"anilist"`
	Filename   string      `json:"filename"`
	Episode    any         `json:"episode"`
	Duration   float64     `json:"duration"`
	From       float64     `json:"from"`
	To         float64     `json:"to"`
	At         float64     `json:"at"`
	Similarity float64     `json:"similarity"`
	Video      string      `json:"video"`
	Image      string      `json:"image"`
}

type TraceMoeAPI struct {
	FrameCount int              `json:"frameCount"`
	Error      string           `json:"error"`
	Result     []TraceMoeResult `json:"result"`
}

type AnilistInfo struct {
	ID       int               `json:"id"`
	IDMal    int               `json:"idMal"`
	Title    AnilistInfoTitles `json:"title"`
	Synonyms []string          `json:"synonyms"`
	IsAdult  bool              `json:"isAdult"`
}

type AnilistInfoTitles struct {
	Native  string `json:"native"`
	Romaji  string `json:"romaji"`
	English any    `json:"english"`
}

func fetchTraceMoe(attURL string) ([]byte, error) {
	apiURL := fmt.Sprintf("%s?url=%s&cutBorders&anilistInfo", traceMoeBase, url.QueryEscape(attURL))
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Có lỗi đã xảy ra: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 413 {
		return nil, fmt.Errorf("Lỗi: Video/Ảnh phải nhỏ hơn 25MB %d", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Lỗi: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Có lỗi khi đọc body: %w", err)
	}

	return body, nil
}

func buildTraceMoeAPI(data []byte) (TraceMoeAPI, error) {
	var result TraceMoeAPI

	if err := json.Unmarshal(data, &result); err != nil {
		return TraceMoeAPI{}, fmt.Errorf("lỗi parse JSON: %w", err)
	}

	if result.Error != "" {
		return TraceMoeAPI{}, fmt.Errorf("lỗi: %s", result.Error)
	}

	return result, nil
}
