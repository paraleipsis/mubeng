package runner

import (
	"context"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ktbs.dev/mubeng/common"
	"ktbs.dev/mubeng/internal/bot"
	"ktbs.dev/mubeng/internal/bot/handlers"
	"ktbs.dev/mubeng/internal/checker"
	"ktbs.dev/mubeng/internal/daemon"
	"ktbs.dev/mubeng/internal/server"
	"os"
)

// New to switch an action, whether to check or run a proxy server.
func New(opt *common.Options) error {
	if opt.CheckPeriodically {
		ctx := context.Background()

		proxyChecker := &checker.ProxyChecker{}

		go proxyChecker.Run(opt)

		if opt.TgBot {
			botAPI, err := tgbotapi.NewBotAPI(os.Getenv("TG_BOT_TOKEN"))

			if err != nil {
				return errors.New("failed to create bot api")
			}

			proxyStorage := bot.NewProxyStorage(opt.ProxyManager)
			proxyBot := bot.New(botAPI)

			proxyBot.RegisterCmdView(
				"start",
				handlers.ViewCmdList(),
			)
			proxyBot.RegisterCmdView(
				"online",
				handlers.ViewCmdListLiveProxy(proxyStorage, false),
			)
			proxyBot.RegisterCmdView(
				"offline",
				handlers.ViewCmdListLiveProxy(proxyStorage, true),
			)
			proxyBot.RegisterCmdView(
				"add",
				handlers.ViewCmdAddProxy(proxyStorage),
			)
			proxyBot.RegisterCmdView(
				"delonline",
				handlers.ViewCmdDeleteProxy(proxyStorage, bot.Online),
			)
			proxyBot.RegisterCmdView(
				"deloffline",
				handlers.ViewCmdDeleteProxy(proxyStorage, bot.Offline),
			)
			proxyBot.RegisterCmdView(
				"pruneoffline",
				handlers.ViewCmdPruneOfflineProxy(proxyStorage),
			)

			go proxyBot.Run(ctx)
		}

		<-ctx.Done()

		if opt.Output != "" {
			defer func(Result *os.File) {
				_ = Result.Close()
			}(opt.Result)
		}
	} else if opt.Address != "" {
		if opt.Daemon {
			return daemon.New(opt)
		}

		server.Run(opt)
	} else if opt.Check {
		proxyChecker := &checker.ProxyChecker{}

		proxyChecker.Do(opt)

		if opt.Output != "" {
			defer opt.Result.Close()
		}
	} else {
		return errors.New("no action to run")
	}

	return nil
}
