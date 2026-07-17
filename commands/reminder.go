package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/json"
	"github.com/speedtest002/azunyan-2-go/azunyan2go"
	"github.com/zekroTJA/shinpuru/pkg/hammertime"
)

var reminderCommand = discord.SlashCommandCreate{
	Name:        "remind",
	Description: "Viktor E",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "create",
			Description: "create reminder",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "content",
					Description: "content you want to send",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "time",
					Description: "time to send reminder at, default in UTC+7, ex: `15h04`, `15:04`, `15:04:05`, `3:04PM",
					MinLength:   json.Ptr(3),
					MaxLength:   json.Ptr(8),
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "date",
					Description: "date to send reminder on dd/mm (or dd/mm/yyyy)",
					MaxLength:   json.Ptr(10),
					Required:    false,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "list",
			Description: "list all upcoming reminder(s)",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionBool{
					Name:        "all",
					Description: "true = list all reminder(s) everywhere, (default) false = current channel",
					Required:    false,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "ephemeral",
					Description: "only you can see this message",
					Required:    false,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "cancel",
			Description: "cancel a reminder",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:        "id",
					Description: "reminder's id (use `/remind list` to see)",
					Required:    true,
					MinValue:    json.Ptr(1),
				},
			},
		},
	},
}

func reminderCreateHandler(b *azunyan2go.Bot) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		timeStr, ok := data.OptString("time")
		if !ok {
			return e.CreateMessage(discord.MessageCreate{Content: "`time` is required!"}.WithEphemeral(true))
		}

		dateStr, ok := data.OptString("date")
		if !ok {
			dateStr = ""
		}
		userID := int64(e.User().ID)
		userTimezone, err := b.DB.GetUserLocation(e.Ctx, userID, nil)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("failed to get your timezone: %v", err)}.WithEphemeral(true))
		}
		now := time.Now()
		dueAt, err := ParseDueAt(timeStr, dateStr, userTimezone, now)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("invalid time or date: %v", err)}.WithEphemeral(true))
		}

		channelID := int64(e.Channel().ID())
		var guildID *int64
		if gid := e.GuildID(); gid != nil {
			v := int64(*gid)
			guildID = &v
		}
		content, ok := data.OptString("content")
		if !ok {
			return e.CreateMessage(discord.MessageCreate{Content: "`content` is required!"}.WithEphemeral(true))
		}

		reminderId, err := b.Scheduler.CreateReminder(userID, channelID, guildID, content, dueAt)
		if err != nil {
			return e.CreateMessage(discord.MessageCreate{Content: "there was an error saving your reminder"}.WithEphemeral(true))
		}
		return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("saved! your reminder id is `%v`", reminderId)}.WithEphemeral(true))
	}
}

func reminderCancelHandler(b *azunyan2go.Bot) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		reminderID, ok := data.OptInt("id")
		if !ok {
			return e.CreateMessage(discord.NewMessageCreate().WithContent("id is required!").WithEphemeral(true))
		}
		err := b.Scheduler.CancelReminder(int64(reminderID), int64(e.User().ID))
		if err != nil {
			return e.CreateMessage(discord.NewMessageCreate().WithContent("cancel failed... please try later").WithEphemeral(true))
		}
		return e.CreateMessage(discord.NewMessageCreate().WithContent("cancel succeed!").WithEphemeral(true))

	}
}

func reminderListHandler(b *azunyan2go.Bot) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		userID := int64(e.User().ID)
		isAll, _ := data.OptBool("all")
		ephemeral, _ := data.OptBool("ephemeral")

		var channelID *int64
		if !isAll {
			channelID = json.Ptr(int64(e.Channel().ID()))
		}

		urs, err := b.Scheduler.GetReminders(userID, channelID)
		if err != nil {
			return e.CreateMessage(discord.NewMessageCreate().WithContent("list failed... please try later").WithEphemeral(true))
		}

		if len(urs) == 0 {
			return e.CreateMessage(discord.NewMessageCreate().WithContent("no upcoming reminders").WithEphemeral(ephemeral))
		}

		var idVals, timeVals, contentVals string
		for _, ur := range urs {
			idVals += fmt.Sprintf("%d\n", ur.ID)
			timeVals += fmt.Sprintf("%s\n", hammertime.Format(ur.DueAt, hammertime.Span))
			contentVals += fmt.Sprintf("%s\n", truncateContent(ur.Content, 50))
		}

		embed := discord.NewEmbed().
			WithColor(0x00FFFF).
			WithTitle("your reminder(s)").
			AddField("ID", idVals, true).
			AddField("Due At", timeVals, true).
			AddField("Content", contentVals, true)

		msg := discord.NewMessageCreate().AddEmbeds(embed)
		if ephemeral {
			msg = msg.WithEphemeral(true)
		}
		return e.CreateMessage(msg)
	}
}

func truncateContent(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

var acceptedTimeLayouts = []string{
	"15:04:05",
	"15:04",
	"15h04",
	time.Kitchen, // "3:04PM"
}

func parseTime(timeStr string) (hour, min, sec int, err error) {
	timeStr = strings.TrimSpace(timeStr)
	for _, layout := range acceptedTimeLayouts {
		t, e := time.Parse(layout, timeStr)
		if e == nil {
			return t.Hour(), t.Minute(), t.Second(), nil
		}
	}
	return 0, 0, 0, fmt.Errorf("unrecognized time format %q (try: 15:04, 15:04:05, 15h04, 3:04PM)", timeStr)
}

func parseDate(dateStr string, loc *time.Location, now time.Time) (year int, month time.Month, day int, ok bool, err error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return 0, 0, 0, false, nil
	}

	t, e := time.ParseInLocation("02/01/2006", dateStr, loc)
	if e == nil {
		y, m, d := t.Date()
		return y, m, d, true, nil
	}

	t, e = time.ParseInLocation("02/01", dateStr, loc)
	if e == nil {
		_, m, d := t.Date()
		y := now.In(loc).Year()
		return y, m, d, true, nil
	}

	return 0, 0, 0, false, fmt.Errorf("unrecognized date format %q (try: dd/mm or dd/mm/yyyy)", dateStr)
}

func ParseDueAt(timeStr, dateStr string, loc *time.Location, now time.Time) (time.Time, error) {
	hour, min, sec, err := parseTime(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	nowLocal := now.In(loc)
	year, month, day := nowLocal.Date()

	dateYear, dateMonth, dateDay, ok, err := parseDate(dateStr, loc, now)
	if err != nil {
		return time.Time{}, err
	}
	if ok {
		year, month, day = dateYear, dateMonth, dateDay
	}

	due := time.Date(year, month, day, hour, min, sec, 0, loc)

	if !ok && due.Before(nowLocal) {
		due = due.AddDate(0, 0, 1)
	}

	if due.Before(nowLocal) {
		return time.Time{}, fmt.Errorf("time %s is in the past", due.Format("02/01/2006 15:04:05"))
	}

	return due.UTC(), nil
}
