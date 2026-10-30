package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	interval time.Duration // 30s
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewScheduler(db *db.DB, client bot.Client, interval time.Duration) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:       db,
		client:   client,
		interval: interval,
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
	reminders, err := s.db.GetPendingUserReminders(s.ctx, time.Now().Add(s.interval))
	if err != nil {
		slog.Error("scheduler: failed to load pending reminders", "err", err)
		return
	}
	for _, r := range reminders {
		s.scheduleReminder(r.ID, r.DueAt)
	}
}

func (s *Scheduler) scheduleReminder(reminderID int64, dueAt time.Time) {
	d := time.Until(dueAt)
	if d <= 0 {
		s.claimAndSend(reminderID)
		return
	}
	time.AfterFunc(d, func() {
		s.claimAndSend(reminderID)
	})
}

func (s *Scheduler) claimAndSend(reminderID int64) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	userID, message, channelID, dueAt, claimed, err := s.db.ClaimUserReminder(ctx, reminderID)
	if err != nil {
		slog.Error("scheduler: failed to claim reminder", "reminder_id", reminderID, "err", err)
		return
	}
	if !claimed {
		return
	}

	embed := discord.NewEmbed().
		WithDescription(message).
		WithColor(0x00FFFF).
		WithFooterText(fmt.Sprintf("ID: %d - %s", reminderID, hammertime.Format(dueAt, hammertime.Span)))

	var usersGetRemind strings.Builder
	usersGetRemind.WriteString(fmt.Sprintf("<@%d>", userID)) // reminder's owner

	usersIDGetRemind, err := s.db.GetUsersReminderPing(ctx, reminderID)
	if err != nil {
		slog.Error("scheduler: failed to get reminder pings", "reminder_id", reminderID, "err", err)
	} else {
		for _, userIDGetRemind := range usersIDGetRemind {
			if userIDGetRemind != userID {
				usersGetRemind.WriteString(fmt.Sprintf(", <@%d>", userIDGetRemind))
			}
		}
	}

	_, err = s.client.Rest.CreateMessage(
		snowflake.ID(channelID),
		discord.NewMessageCreate().
			WithContent(usersGetRemind.String()).
			AddEmbeds(embed).
			WithAllowedMentions(&discord.AllowedMentions{Parse: []discord.AllowedMentionType{discord.AllowedMentionTypeUsers}}),
	)
	if err != nil {
		slog.Error("scheduler: failed to send reminder message", "reminder_id", reminderID, "channel_id", channelID, "user_id", userID, "err", err)
		if err = s.db.UpdateStatusUserReminder(ctx, reminderID, "FAILED"); err != nil {
			slog.Error("scheduler: failed to update reminder status", "reminder_id", reminderID, "status", "FAILED", "err", err)
		}
		return
	}

	if err = s.db.UpdateStatusUserReminder(ctx, reminderID, "SENT"); err != nil {
		slog.Error("scheduler: failed to update reminder status", "reminder_id", reminderID, "status", "SENT", "err", err)
	}
}

func (s *Scheduler) CreateReminder(ctx context.Context, userID int64, channelID int64, guildID *int64, message string, dueAt time.Time) (id int64, err error) {
	id, err = s.db.CreateUserReminder(ctx, userID, channelID, guildID, message, dueAt)
	if err != nil {
		return 0, fmt.Errorf("service.CreateReminder: %w", err)
	}
	if time.Until(dueAt) <= (s.interval - 1*time.Second) {
		s.scheduleReminder(id, dueAt)
	}
	return id, nil
}

func (s *Scheduler) Shutdown() {
	s.cancel()
	s.wg.Wait()
}
