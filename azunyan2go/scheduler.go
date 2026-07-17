package azunyan2go

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/speedtest002/azunyan-2-go/db"
	"github.com/zekroTJA/shinpuru/pkg/hammertime"
)

type Scheduler struct {
    db       *db.DB
	client   bot.Client
    interval time.Duration   // 30s
    window   time.Duration   // interval * 3 (90s)
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
}

func NewScheduler(db *db.DB, client bot.Client, interval time.Duration) *Scheduler {
    ctx, cancel := context.WithCancel(context.Background())
    return &Scheduler{
        db:       db,
		client: client,
        interval: interval,
        window:   interval * 3,
        ctx:      ctx,
        cancel:   cancel,
    }
}

func (s *Scheduler) Start() {
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.loadAndSchedule()
        
        ticker := time.NewTicker(s.interval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                s.loadAndSchedule()
            case <-s.ctx.Done():
                return
            }
        }
    }()
}

func (s *Scheduler) loadAndSchedule() {
    now := time.Now()
    reminders, err := s.db.GetPendingUserReminders(s.ctx, now, now.Add(s.window))
	if err != nil {
		slog.Error("failed when load pending reminder", "err", err)
		return
	}
    for _, r := range reminders {
        s.scheduleReminder(r.ID, r.DueAt)
    }
}

func (s *Scheduler) scheduleReminder(id int64, dueAt time.Time) {
	d := time.Until(dueAt)
	if d <= 0 {
		s.claimAndSend(id)
		return
	}
	time.AfterFunc(d,  func() {
		s.claimAndSend(id)
	})
}

func (s *Scheduler) claimAndSend(id int64) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	userID, message, channelID, dueAt, claimed, err := s.db.ClaimUserReminder(ctx, id)
	if err != nil {
		slog.Error("failed to claim remider", "error", err)
		return
	}
	if !claimed {
		return
	}

	embed := discord.NewEmbed().
		WithDescription(message).
		WithColor(0x00FFFF).
		WithFooterText(fmt.Sprintf("ID: %d - %s", id, hammertime.Format(dueAt, hammertime.Span)))

	_, err = s.client.Rest.CreateMessage(
		snowflake.ID(channelID),
		discord.NewMessageCreate().
			WithContent(fmt.Sprintf("<@%d>", userID)).
			AddEmbeds(embed).
			WithAllowedMentions(&discord.AllowedMentions{Parse: []discord.AllowedMentionType{discord.AllowedMentionTypeUsers}}),
	)
	if err != nil {
		slog.Error("failed to send reminder", "reminder_id", id, "channel_id", channelID, "err", err)
		if err = s.db.UpdateStatusUserReminder(ctx, id, "FAILED"); err != nil {
			slog.Error("failed to mark reminder as FAILED", "reminder_id", id, "err", err)
		}
		return
	}

	if err = s.db.UpdateStatusUserReminder(ctx, id, "SENT"); err != nil {
		slog.Error("failed to mark reminder as SENT", "reminder_id", id, "err", err)
	}
}

func (s *Scheduler) CreateReminder(userID int64, channelID int64, guildID *int64, message string, dueAt time.Time) (id int64, err error) {
	id, err = s.db.CreateUserReminder(s.ctx, userID, channelID, guildID, message, dueAt)
	if err != nil {
		slog.Error("failed to create reminder", "reminder_id", id, "err", err)
	}
	if time.Until(dueAt) <= (s.interval - 1 * time.Second) {
		s.scheduleReminder(id, dueAt)
	} 
	return id, err
}

func (s *Scheduler) CancelReminder(reminderID, userID int64) (err error) {
	err = s.db.CancelUserReminder(s.ctx, reminderID, userID)
	if err != nil {
		slog.Error("failed to cancel reminder", "reminder_id", reminderID, "err", err)
	}
	return err
}

func (s *Scheduler) GetReminders(userID int64, channelID *int64) (ur []db.UserReminder ,err error) {
	ur, err = s.db.GetUserReminders(s.ctx, userID, channelID)
	if err != nil {
		slog.Error("failed to get reminders", "error", err)
		return nil, err
	}
	return ur, nil
}

func (s *Scheduler) Shutdown() {
    s.cancel()
    s.wg.Wait()
}