package checker

import (
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"github.com/robfig/cron/v3"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/sourcegraph/conc/pool"
	"ktbs.dev/mubeng/common"
	"ktbs.dev/mubeng/pkg/helper"
	"ktbs.dev/mubeng/pkg/mubeng"
)

type ProxyChecker struct {
	lastTgMsgIDs []int
}

type SendTgMsgResponse struct {
	Result struct {
		MessageId *int `json:"message_id,omitempty"`
	} `json:"result"`
}

func (pc *ProxyChecker) Run(opt *common.Options) {
	c := cron.New()

	_, err := c.AddFunc(opt.PollingPeriod, func() {
		pc.Do(opt)
	},
	)

	if err != nil {
		gologger.Fatal().Msgf("Error! %s", err)
	}

	c.Start()

	watcher, err := opt.ProxyManager.Watch()

	if err != nil {
		gologger.Fatal().Msgf("Error! %s", err)
	}

	go opt.ProxyManager.WatchFile(watcher)
}

// Do checks proxy from list.
//
// Displays proxies that have died if verbose mode is enabled,
// or save live proxies into user defined files.
func (pc *ProxyChecker) Do(opt *common.Options) {
	p := pool.New().WithMaxGoroutines(opt.Goroutine)
	var diedProxies []string
	var liveProxies []IPInfo
	var liveProxiesAddresses []string

	for _, proxy := range opt.ProxyManager.Proxies {
		address := helper.EvalFunc(proxy)

		p.Go(func() {
			addr, err := pc.check(address, opt.Timeout)

			if len(opt.Countries) > 0 && !pc.isMatchCC(opt.Countries, addr.Country) {
				return
			}

			if err != nil {
				if opt.Verbose {
					fmt.Printf("[%s] %s\n", aurora.Red("DIED"), address)
				}

				diedProxies = append(diedProxies, address)
			} else {
				fmt.Printf("[%s] [%s] [%s] %s\n", aurora.Green("LIVE"), aurora.Magenta(addr.Country), aurora.Cyan(addr.IP), address)
				addr.IP = address

				liveProxies = append(liveProxies, addr)

				liveProxiesAddresses = append(liveProxiesAddresses, address)
			}
		})
	}

	p.Wait()

	if opt.Output != "" {
		var proxies []byte

		for _, ipInfo := range liveProxies {
			if ipInfo.Country != "" {
				proxies = append(proxies, []byte(fmt.Sprintf("%s|%s|%s\n", ipInfo.IP, ipInfo.Country, strings.ReplaceAll(ipInfo.City, " ", "")))...)
			} else {
				proxies = append(proxies, []byte(fmt.Sprintf("%s\n", ipInfo.IP))...)
			}
		}

		err := os.WriteFile(opt.Result.Name(), proxies, 0644)

		if err != nil {
			gologger.Error().Msgf("Error! %s", err)
			return
		}
	}

	opt.ProxyManager.LiveProxies = liveProxiesAddresses
	opt.ProxyManager.DiedProxies = diedProxies

	if len(diedProxies) > 0 {
		if opt.TgAlert {
			pc.handleTgAlert(diedProxies)
		}
	} else {
		pc.pruneLastAlerts()
	}
}

func (pc *ProxyChecker) isMatchCC(cc []string, code string) bool {
	if code == "" {
		return false
	}

	for _, c := range cc {
		if code == strings.ToUpper(strings.TrimSpace(c)) {
			return true
		}
	}

	return false
}

func (pc *ProxyChecker) check(address string, timeout time.Duration) (IPInfo, error) {
	var info IPInfo

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return info, err
	}

	tr, err := mubeng.Transport(address)
	if err != nil {
		return info, err
	}

	proxy := &mubeng.Proxy{
		Address:   address,
		Transport: tr,
	}

	client, req = proxy.New(req)
	client.Timeout = timeout
	req.Header.Add("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return info, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(body, &info)
	if err != nil {
		return info, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	defer tr.CloseIdleConnections()

	return info, nil
}
