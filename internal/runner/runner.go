package runner

import (
	"context"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ktbs.dev/mubeng/common"
	"ktbs.dev/mubeng/internal/bot"
	"ktbs.dev/mubeng/internal/bot/handlers"
	"ktbs.dev/mubeng/internal/bot/middleware"
	"ktbs.dev/mubeng/internal/checker"
	"ktbs.dev/mubeng/internal/daemon"
	"ktbs.dev/mubeng/internal/server"
	"os"
	"strconv"
	"strings"
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

			var intUsersFilter []int64
			usersFilterEnv := os.Getenv("USERS_FILTER")

			if usersFilterEnv != "" {
				usersFilterStr := strings.Trim(usersFilterEnv, "[]")
				usersFilter := strings.Split(usersFilterStr, ",")

				for i, v := range usersFilter {
					if v == "" {
						continue
					}

					v = strings.ReplaceAll(v, " ", "")

					id, err := strconv.ParseInt(v, 10, 64)

					if err != nil {
						return err
					}

					intUsersFilter[i] = id
				}
			}

			proxyBot.RegisterCmdView(
				"start",
				middleware.UsersFilter(
					handlers.ViewCmdList(),
					intUsersFilter,
				),
			)
			proxyBot.RegisterCmdView(
				"online",
				middleware.UsersFilter(
					handlers.ViewCmdListLiveProxy(proxyStorage, false),
					intUsersFilter,
				),
			)
			proxyBot.RegisterCmdView(
				"offline",
				middleware.UsersFilter(
					handlers.ViewCmdListLiveProxy(proxyStorage, true),
					intUsersFilter,
				),
			)
			proxyBot.RegisterCmdView(
				"add",
				middleware.UsersFilter(
					handlers.ViewCmdAddProxy(proxyStorage),
					intUsersFilter,
				),
			)
			proxyBot.RegisterCmdView(
				"delonline",
				middleware.UsersFilter(
					handlers.ViewCmdDeleteProxy(proxyStorage, bot.Online),
					intUsersFilter,
				),
			)
			proxyBot.RegisterCmdView(
				"deloffline",
				middleware.UsersFilter(
					handlers.ViewCmdDeleteProxy(proxyStorage, bot.Offline),
					intUsersFilter,
				),
			)
			proxyBot.RegisterCmdView(
				"pruneoffline",
				middleware.UsersFilter(
					handlers.ViewCmdPruneOfflineProxy(proxyStorage),
					intUsersFilter,
				),
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
