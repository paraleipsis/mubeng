package handlers

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ktbs.dev/mubeng/internal/bot"
)

type ProxyPruneStorage interface {
	DeleteProxies(ctx context.Context, proxies ...string) error
	GetOfflineProxies(ctx context.Context) ([]string, error)
}

func ViewCmdPruneOfflineProxy(storage ProxyPruneStorage) bot.ViewFunc {
	return func(ctx context.Context, botAPI *tgbotapi.BotAPI, update tgbotapi.Update) error {
		proxiesList, err := storage.GetOfflineProxies(ctx)

		if err != nil {
			return err
		}

		if len(proxiesList) == 0 {
			if _, err = botAPI.Send(
				tgbotapi.NewMessage(update.Message.Chat.ID, "No proxies to delete"),
			); err != nil {
				return err
			}

			return nil
		}

		err = storage.DeleteProxies(ctx, proxiesList...)

		if err != nil {
			return err
		}

		msgText := "Offline proxies have been pruned from monitoring list"
		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

		reply.ParseMode = bot.ParseModeMarkdownV2

		if _, err = botAPI.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
