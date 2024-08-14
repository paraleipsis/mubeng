package handlers

import (
	"context"
	"fmt"
	"ktbs.dev/mubeng/internal/bot"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ViewCmdList() bot.ViewFunc {
	return func(ctx context.Context, botAPI *tgbotapi.BotAPI, update tgbotapi.Update) error {
		opts := []string{
			"/online - Get online proxies list",
			"/offline - Get offline proxies list",
			"/add - Add proxies. Separate multiple proxies with a comma. For example proxy1,proxy2.",
			"/delonline - Delete online proxies from monitoring list by IDs. Separate multiple proxies with a comma. For example 1,2,3,4.",
			"/deloffline - Delete offline proxies from monitoring list by IDs. Separate multiple proxies with a comma. For example 1,2,3,4.",
			"/pruneoffline - Delete all offline proxies from monitoring list",
		}

		msgText := fmt.Sprintf(
			"Options:\n\n%s",
			strings.Join(opts, "\n"),
		)

		msgText = bot.EscapeForMarkdown(msgText)

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		reply.ParseMode = bot.ParseModeMarkdownV2

		if _, err := botAPI.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
