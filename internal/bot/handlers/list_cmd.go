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
			"/all - Get all proxies list\n",
			"/online - Get online proxies list\n",
			"/offline - Get offline proxies list\n",

			"/addhttp - Add http proxies. Separate multiple proxies with a space/comma/CRLF. For example ip:port:user:password ip:port:user:password.\n",
			"/addhttps - Add https proxies. Separate multiple proxies with a space/comma/CRLF. For example ip:port:user:password ip:port:user:password.\n",

			"/delonline - Delete online proxies from monitoring list by IDs. Separate multiple proxies with a space. For example 1 2 3 4.\n",
			"/deloffline - Delete offline proxies from monitoring list by IDs. Separate multiple proxies with a space. For example 1 2 3 4.\n",
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
