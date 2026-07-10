package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var sleepCommand = discord.SlashCommandCreate{
	Name:        "sleep",
	Description: "a mimir",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name:        "wakeup",
			Description: "support `12:00AM` or `23:59`",
			Required:    false,
		},
	},
}

func SleepHandler(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	timeOpt, _ := data.OptString("wakeup")
	msg, err := SuggestSleepTimes(timeOpt)
	if err != nil {
		return e.CreateMessage(
			discord.MessageCreate{
				Content: fmt.Sprintf("Có lỗi xảy ra:\n%v", err),
			}.WithEphemeral(true),
		)
	}
	return e.CreateMessage(
		discord.MessageCreate{
			Content: msg,
		},
	)
}

const (
	cycleLength    = 90 * time.Minute
	fallAsleepTime = 15 * time.Minute
	maxCycles      = 6
	gifURL         = "https://tenor.com/view/onimai-anime-sleep-mihari-oyama-gif-27609322"
)

type SleepEntry struct {
	Time   string
	Cycles int
	Hours  float64
}

func parseFlexibleTime(raw string) (time.Time, error) {
	for _, layout := range []string{time.Kitchen, "15:04"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("không parse được %q", raw)
}

func buildEntries(anchor time.Time, forward bool) []SleepEntry {
	entries := make([]SleepEntry, maxCycles)
	for i := 1; i <= maxCycles; i++ {
		offset := time.Duration(i)*cycleLength + fallAsleepTime
		t := anchor.Add(offset)
		if !forward {
			t = anchor.Add(-offset)
		}

		idx := i - 1
		if !forward {
			idx = maxCycles - i
		}
		entries[idx] = SleepEntry{
			Time:   t.Format(time.Kitchen),
			Cycles: i,
			Hours:  float64(i) * 1.5,
		}
	}
	return entries
}

func buildResponse(entries []SleepEntry, mode, wakeupArg string, currentTime time.Time) string {
	var sb strings.Builder

	if mode == "wakeup" {
		sb.WriteString(fmt.Sprintf(
			"Bây giờ là **%s**. Nếu bạn đi ngủ ngay bây giờ, bạn nên cố gắng thức dậy vào một trong những thời điểm sau:\n\n",
			currentTime.Format("03:04 PM"),
		))
	} else {
		sb.WriteString(fmt.Sprintf(
			"Nếu bạn muốn tỉnh giấc vào **%s**, bạn nên cố gắng đi vào giấc ngủ vào một trong những thời điểm sau:\n\n",
			wakeupArg,
		))
	}

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("⏰ **%s** cho %d chu kỳ - ngủ %.1f tiếng.\n", e.Time, e.Cycles, e.Hours))
	}

	sb.WriteString("\nXin lưu ý rằng bạn nên đi ngủ vào những thời điểm này. Con người trung bình mất ~14 phút để đi vào giấc ngủ, vì vậy hãy lên kế hoạch cho phù hợp!\n\nChúc ngủ ngon! 😴")
	sb.WriteString(fmt.Sprintf("\n[zzz](%s)", gifURL))

	return sb.String()
}

func SuggestSleepTimes(rawTime string) (string, error) {
	now := time.Now()

	if rawTime == "" {
		entries := buildEntries(now, true)
		return buildResponse(entries, "wakeup", "", now), nil
	}

	wakeTime, err := parseFlexibleTime(rawTime)
	if err != nil {
		return "", fmt.Errorf("không parse được giờ đầu vào: %w", err)
	}
	entries := buildEntries(wakeTime, false)
	return buildResponse(entries, "bedtime", rawTime, now), nil
}
