package commands

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/handler"
)

func LoggingMiddleware(next handler.Handler) handler.Handler {
	return func(e *handler.InteractionEvent) error {
		i := e.Interaction
		user := i.User()
		guildID := i.GuildID()

		start := time.Now()
		err := next(e)

		attrs := []any{
			slog.String("user_id", user.ID.String()),
			slog.String("username", user.Tag()),
			slog.String("interaction_type", fmt.Sprint(i.Type())),
			slog.Duration("duration", time.Since(start)),
		}

		if guildID != nil {
			attrs = append(attrs, slog.String("guild_id", guildID.String()))
		}

		if err != nil {
			attrs = append(attrs, slog.Any("err", err))
			slog.Error("interaction failed", attrs...)
		} else {
			slog.Info("interaction handled", attrs...)
		}

		return err
	}
}
