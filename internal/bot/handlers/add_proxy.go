package handlers

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ktbs.dev/mubeng/internal/bot"
	"net"
	"net/url"
	"strings"
)

type ProxyStorage interface {
	AddProxies(ctx context.Context, proxies ...string) error
}

func ViewCmdAddProxy(storage ProxyStorage) bot.ViewFunc {
	return func(ctx context.Context, botAPI *tgbotapi.BotAPI, update tgbotapi.Update) error {
		proxies := update.Message.CommandArguments()
		proxiesList := strings.Split(proxies, ",")

		for _, proxy := range proxiesList {
			parsedURL, err := url.Parse(proxy)

			if err != nil {
				if _, err = botAPI.Send(
					tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("invalid URL: %s", proxy)),
				); err != nil {
					return err
				}

				return nil
			}

			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				if _, err = botAPI.Send(
					tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("invalid URL: %s", proxy)),
				); err != nil {
					return err
				}

				return nil
			}

			ip := net.ParseIP(parsedURL.Hostname())

			if ip == nil {
				if _, err = botAPI.Send(
					tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("invalid IP address: %s", proxy)),
				); err != nil {
					return err
				}

				return nil
			}

			if parsedURL.Port() == "" {
				if _, err = botAPI.Send(
					tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("missing port in URL: %s", proxy)),
				); err != nil {
					return err
				}

				return nil
			}

			if _, err = net.LookupPort("tcp", parsedURL.Port()); err != nil {
				if _, err = botAPI.Send(
					tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("invalid port: %s", proxy)),
				); err != nil {
					return err
				}

				return nil
			}
		}

		err := storage.AddProxies(ctx, proxiesList...)

		if err != nil {
			return err
		}

		msgText := "Proxies have been added to monitoring list"
		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

		reply.ParseMode = bot.ParseModeMarkdownV2

		if _, err = botAPI.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
