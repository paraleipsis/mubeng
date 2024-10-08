package handlers

import (
	"context"
	"fmt"
	"ktbs.dev/mubeng/internal/bot"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ProxyLister interface {
	GetOnlineProxies(ctx context.Context) ([]string, error)
	GetOfflineProxies(ctx context.Context) ([]string, error)
	GetAllProxies(ctx context.Context) ([]string, error)
}

func ViewCmdListProxy(lister ProxyLister, status bot.ProxyStatus) bot.ViewFunc {
	return func(ctx context.Context, botAPI *tgbotapi.BotAPI, update tgbotapi.Update) error {
		var proxies []string
		var err error
		var msgText string
		var statusMsg string

		switch status {
		case bot.Offline:
			proxies, err = lister.GetOfflineProxies(ctx)

			if err != nil {
				return err
			}

			statusMsg = "Offline"
		case bot.Online:
			proxies, err = lister.GetOnlineProxies(ctx)

			if err != nil {
				return err
			}

			statusMsg = "Online"
		case bot.All:
			proxies, err = lister.GetAllProxies(ctx)

			if err != nil {
				return err
			}

			statusMsg = "All"
		}

		msgProxies := make([]string, 0)

		for i, p := range proxies {
			address := p
			msgProxies = append(msgProxies, fmt.Sprintf("%d. %s", i+1, address))
		}

		msgText = fmt.Sprintf(
			"%s Proxies (total %d):\n\n%s",
			statusMsg,
			len(msgProxies),
			strings.Join(msgProxies, "\n"),
		)

		msgText = bot.EscapeForMarkdown(msgText)

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		reply.ParseMode = bot.ParseModeMarkdownV2

		if _, err = botAPI.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
