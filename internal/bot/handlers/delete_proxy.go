package handlers

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ktbs.dev/mubeng/internal/bot"
	"strconv"
	"strings"
)

type ProxyDeleteStorage interface {
	DeleteProxies(ctx context.Context, proxies ...string) error
	GetOnlineProxies(ctx context.Context) ([]string, error)
	GetOfflineProxies(ctx context.Context) ([]string, error)
}

func ViewCmdDeleteProxy(storage ProxyDeleteStorage, proxyStatus bot.ProxyStatus) bot.ViewFunc {
	return func(ctx context.Context, botAPI *tgbotapi.BotAPI, update tgbotapi.Update) error {
		proxiesIds := update.Message.CommandArguments()
		proxiesIdsStr := strings.Split(proxiesIds, ",")

		var proxiesToDelete []string
		var proxiesList []string
		var err error

		switch proxyStatus {
		case bot.Online:
			proxiesList, err = storage.GetOnlineProxies(ctx)

			if err != nil {
				return err
			}
		case bot.Offline:
			proxiesList, err = storage.GetOfflineProxies(ctx)

			if err != nil {
				return err
			}
		}

		if len(proxiesList) == 0 {
			if _, err = botAPI.Send(
				tgbotapi.NewMessage(update.Message.Chat.ID, "No proxies to delete"),
			); err != nil {
				return err
			}

			return nil
		}

		for _, proxyId := range proxiesIdsStr {
			if i, err := strconv.ParseInt(proxyId, 10, 64); err != nil {
				if _, err = botAPI.Send(
					tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("invalid ID: %s", proxyId)),
				); err != nil {
					return err
				}

				return nil
			} else {
				proxiesToDelete = append(proxiesToDelete, proxiesList[i-1])
			}
		}

		err = storage.DeleteProxies(ctx, proxiesToDelete...)

		if err != nil {
			return err
		}

		msgText := "Proxies have been deleted from monitoring list"
		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

		reply.ParseMode = bot.ParseModeMarkdownV2

		if _, err = botAPI.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
