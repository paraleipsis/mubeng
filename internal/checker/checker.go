package checker

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/projectdiscovery/gologger"
	"github.com/robfig/cron/v3"
	"io"
	"net"
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
}

// Do checks proxy from list.
//
// Displays proxies that have died if verbose mode is enabled,
// or save live proxies into user defined files.
func (pc *ProxyChecker) Do(opt *common.Options) {
	p := pool.New().WithMaxGoroutines(opt.Goroutine)
	var deadProxies []string
	var liveProxies []IPInfo

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

				unavailableProxy := proxy

				parts := strings.Split(unavailableProxy, "@")

				if len(parts) > 1 {
					unavailableProxy = parts[1]
				}

				deadProxies = append(deadProxies, unavailableProxy)
			} else {
				fmt.Printf("[%s] [%s] [%s] %s\n", aurora.Green("LIVE"), aurora.Magenta(addr.Country), aurora.Cyan(addr.IP), address)
				addr.IP = address

				liveProxies = append(liveProxies, addr)
			}
		})
	}

	p.Wait()

	if opt.Output != "" {
		file, err := os.Open(opt.Result.Name())

		if err != nil {
			gologger.Error().Msgf("Error! %s", err)
			return
		}

		defer func(file *os.File) {
			_ = file.Close()
		}(file)

		var proxies []byte

		for _, ipInfo := range liveProxies {
			if ipInfo.Country != "" {
				proxies = append(proxies, []byte(fmt.Sprintf("%s|%s\n", ipInfo.IP, ipInfo.Country))...)
			} else {
				proxies = append(proxies, []byte(fmt.Sprintf("%s\n", ipInfo.IP))...)
			}
		}

		err = os.WriteFile(opt.Result.Name(), proxies, 0644)

		if err != nil {
			gologger.Error().Msgf("Error! %s", err)
			return
		}
	}

	if len(deadProxies) > 0 && opt.TgAlert {
		msgID, err := pc.sendTgProxyAlert(deadProxies)

		if err != nil {
			gologger.Error().Msgf("Error! %s", err)
			return
		}

		if len(pc.lastTgMsgIDs) != 0 {
			var deletedMsgs []int

			for i, m := range pc.lastTgMsgIDs {
				err = pc.deleteTgMsg(m)

				if err != nil {
					gologger.Error().Msgf("Error! %s", err)
				}

				deletedMsgs = append(deletedMsgs, i)
			}

			for _, d := range deletedMsgs {
				pc.lastTgMsgIDs = append(pc.lastTgMsgIDs[:d], pc.lastTgMsgIDs[d+1:]...)
			}
		}

		pc.lastTgMsgIDs = append(pc.lastTgMsgIDs, *msgID)
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

func (pc *ProxyChecker) newHttpClient() *resty.Client {
	t := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: dialContextTimeout}).DialContext,
		TLSHandshakeTimeout: clientTLSHandshakeTimeout,
	}

	client := resty.New().
		SetDebug(httpClientDebug).
		SetTimeout(clientTimeout).
		SetRetryCount(retryCount).
		SetRetryWaitTime(clientRetryWaitTime).
		SetTransport(t)

	return client
}

func (pc *ProxyChecker) sendTgProxyAlert(proxies []string) (*int, error) {
	restyClient := pc.newHttpClient()

	var textProxies string

	for i, item := range proxies {
		if i == 0 {
			textProxies += "copy%0A"
		}

		if i > 0 {
			textProxies += "%0A"
		}

		item = strings.ReplaceAll(item, ".", "\\.")

		textProxies += fmt.Sprintf("%s", item)
	}

	textToSend := fmt.Sprintf("Unavailable proxies: ```%s```", textProxies)
	urlSendMsg := fmt.Sprintf("%s/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=MarkdownV2", tgAPI, os.Getenv("TG_BOT_TOKEN"), os.Getenv("TG_BOT_CHAT"), textToSend)

	sendResp := &SendTgMsgResponse{}

	resp, err := restyClient.R().SetResult(sendResp).Post(urlSendMsg)

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, UnsuccessfulRequestError
	}

	return sendResp.Result.MessageId, nil
}

func (pc *ProxyChecker) deleteTgMsg(msgID int) error {
	restyClient := pc.newHttpClient()

	urlSendMsg := fmt.Sprintf("%s/bot%s/deleteMessage?chat_id=%s&message_id=%d", tgAPI, os.Getenv("TG_BOT_TOKEN"), os.Getenv("TG_BOT_CHAT"), msgID)

	resp, err := restyClient.R().Post(urlSendMsg)

	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return UnsuccessfulRequestError
	}

	return nil
}
