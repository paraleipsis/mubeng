package middleware

import (
	"context"
	"ktbs.dev/mubeng/internal/bot"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func UsersFilter(next bot.ViewFunc, users []int64) bot.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		if slices.Contains(users, update.SentFrom().ID) || len(users) == 0 {
			return next(ctx, bot, update)
		}

		if _, err := bot.Send(tgbotapi.NewMessage(
			update.FromChat().ID,
			"Permission denied",
		)); err != nil {
			return err
		}

		return nil
	}
}
