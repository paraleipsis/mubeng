package checker

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/projectdiscovery/gologger"
	"net"
	"net/http"
	"os"
	"strings"
)

func (pc *ProxyChecker) handleTgAlert(deadProxies []string) {
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

func (pc *ProxyChecker) newRestyClient() *resty.Client {
	t := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: dialContextTimeout}).DialContext,
		TLSHandshakeTimeout: clientTLSHandshakeTimeout,
	}

	restyClient := resty.New().
		SetDebug(httpClientDebug).
		SetTimeout(clientTimeout).
		SetRetryCount(retryCount).
		SetRetryWaitTime(clientRetryWaitTime).
		SetTransport(t)

	return restyClient
}

func (pc *ProxyChecker) sendTgProxyAlert(proxies []string) (*int, error) {
	restyClient := pc.newRestyClient()

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

	textToSend := fmt.Sprintf("Offline proxies: ```%s```", textProxies)
	urlSendMsg := fmt.Sprintf(
		"%s/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=MarkdownV2",
		tgAPI,
		os.Getenv("TG_BOT_TOKEN"),
		os.Getenv("TG_BOT_CHAT"),
		textToSend,
	)

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
	restyClient := pc.newRestyClient()

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
