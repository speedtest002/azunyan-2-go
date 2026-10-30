package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func LoggingMiddleware(next handler.Handler) handler.Handler {
	return func(e *handler.InteractionEvent) error {
		i := e.Interaction
		user := i.User()
		guildID := i.GuildID()

		cmdName := ""
		var paramAttrs []any
		if cmd, ok := i.(discord.ApplicationCommandInteraction); ok {
			cmdName = cmd.Data.CommandName()
			if slashData, ok := cmd.Data.(discord.SlashCommandInteractionData); ok {
				cmdName = slashData.CommandPath()

				if slashData.SubCommandGroupName != nil {
					paramAttrs = append(paramAttrs, slog.String("sub_group", *slashData.SubCommandGroupName))
				}
				if slashData.SubCommandName != nil {
					paramAttrs = append(paramAttrs, slog.String("sub_command", *slashData.SubCommandName))
				}

				for _, opt := range slashData.All() {
					var val string
					switch opt.Type {
					case discord.ApplicationCommandOptionTypeString:
						val = opt.String()
					case discord.ApplicationCommandOptionTypeInt:
						val = strconv.Itoa(opt.Int())
					case discord.ApplicationCommandOptionTypeBool:
						val = strconv.FormatBool(opt.Bool())
					case discord.ApplicationCommandOptionTypeFloat:
						val = strconv.FormatFloat(opt.Float(), 'f', -1, 64)
					case discord.ApplicationCommandOptionTypeUser,
						discord.ApplicationCommandOptionTypeChannel,
						discord.ApplicationCommandOptionTypeRole,
						discord.ApplicationCommandOptionTypeMentionable:
						val = opt.Snowflake().String()
					default:
						val = string(opt.Value)
					}
					paramAttrs = append(paramAttrs, slog.String("param_"+opt.Name, val))
				}
			}
		}

		start := time.Now()
		err := next(e)

		attrs := []any{
			slog.String("user_id", user.ID.String()),
			slog.String("username", user.Tag()),
			slog.String("interaction_type", fmt.Sprint(i.Type())),
			slog.Duration("duration", time.Since(start)),
		}
		if cmdName != "" {
			attrs = append(attrs, slog.String("command", cmdName))
		}

		if guildID != nil {
			attrs = append(attrs, slog.String("guild_id", guildID.String()))
		}

		attrs = append(attrs, paramAttrs...)

		if err != nil {
			attrs = append(attrs, slog.Any("err", err))
			slog.Error("interaction failed", attrs...)
		} else {
			slog.Info("interaction handled", attrs...)
		}

		return err
	}
}
