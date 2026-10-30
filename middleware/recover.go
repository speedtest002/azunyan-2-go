package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func Recover(next handler.Handler) handler.Handler {
	return func(e *handler.InteractionEvent) (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered in interaction",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				_ = e.CreateMessage(discord.NewMessageCreate().
					WithContent("an internal server error occurred, please try again later").//  Đã có lỗi hệ thống xảy ra, vui lòng thử lại sau.").
					WithEphemeral(true))
			}
		}()
		return next(e)
	}
}
