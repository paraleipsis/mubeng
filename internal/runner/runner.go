package runner

import (
	"context"
	"errors"
	"ktbs.dev/mubeng/common"
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
