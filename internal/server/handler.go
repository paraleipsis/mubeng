package server

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/elazarl/goproxy"
	"io"
	"ktbs.dev/mubeng/common"
	"ktbs.dev/mubeng/pkg/helper"
	"ktbs.dev/mubeng/pkg/mubeng"
	"net/http"
	"os"
	"strings"
)

// onRequest handles client request
func (p *Proxy) onRequest(req *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if p.Options.Sync {
		mutex.Lock()
		defer mutex.Unlock()
	}

	// Rotate proxy IP for every AFTER request
	if (rotate == "") || (ok >= p.Options.Rotate) {
		if p.Options.Method == "sequent" {
			rotate = p.Options.ProxyManager.NextProxy()
		}

		if p.Options.Method == "random" {
			rotate = p.Options.ProxyManager.RandomProxy()
		}

		if p.Options.Method == "round-robin" {
			var rotateOk bool

			rotate, rotateOk = p.Options.ProxyManager.RoundRobin.Next()

			if !rotateOk {
				log.Errorf("%s %s", req.RemoteAddr, "no available proxies")
				resp := goproxy.NewResponse(req, mime, http.StatusBadGateway, "Proxy server error")

				return req, resp
			}

			log.Debugf(rotate)
		}

		if ok >= p.Options.Rotate {
			ok = 1
		}
	} else {
		ok++
	}

	rotate = helper.EvalFunc(rotate)
	resChan := make(chan interface{})

	go func(r *http.Request) {
		if (r.URL.Scheme != "http") && (r.URL.Scheme != "https") {
			resChan <- fmt.Errorf("Unsupported protocol scheme: %s", r.URL.Scheme)
			return
		}

		log.Debugf("%s %s %s", r.RemoteAddr, r.Method, r.URL)

		tr, err := mubeng.Transport(rotate)
		if err != nil {
			resChan <- err
			return
		}

		proxy := &mubeng.Proxy{
			Address:   rotate,
			Transport: tr,
		}

		client, req = proxy.New(req)
		client.Timeout = p.Options.Timeout
		if p.Options.Verbose {
			client.Transport = dump.RoundTripper(tr)
		}

		resp, err := client.Do(req)
		if err != nil {
			resChan <- err
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			resChan <- err
			return
		}

		resp.Body = io.NopCloser(bytes.NewBuffer(buf))

		resChan <- resp
	}(req)

	var resp *http.Response

	res := <-resChan
	switch res := res.(type) {
	case *http.Response:
		resp = res
		log.Debug(req.RemoteAddr, " ", resp.Status)
	case error:
		err := res
		log.Errorf("%s %s", req.RemoteAddr, err)
		resp = goproxy.NewResponse(req, mime, http.StatusBadGateway, "Proxy server error")
	}

	return req, resp
}

// onConnect handles CONNECT method
func (p *Proxy) onConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	if p.Options.Auth != "" {
		auth := ctx.Req.Header.Get("Proxy-Authorization")
		if auth != "" {
			creds := strings.SplitN(auth, " ", 2)
			if len(creds) != 2 {
				return goproxy.RejectConnect, host
			}

			auth, err := base64.StdEncoding.DecodeString(creds[1])
			if err != nil {
				log.Warnf("%s: Error decoding proxy authorization", ctx.Req.RemoteAddr)
				return goproxy.RejectConnect, host
			}

			if string(auth) != p.Options.Auth {
				log.Errorf("%s: Invalid proxy authorization", ctx.Req.RemoteAddr)
				return goproxy.RejectConnect, host
			}
		} else {
			log.Warnf("%s: Unathorized proxy request to %s", ctx.Req.RemoteAddr, host)
			return goproxy.RejectConnect, host
		}
	}

	return goproxy.MitmConnect, host
}

// onResponse handles backend responses, and removing hop-by-hop headers
func (p *Proxy) onResponse(resp *http.Response, _ *goproxy.ProxyCtx) *http.Response {
	for _, h := range mubeng.HopHeaders {
		resp.Header.Del(h)
	}

	return resp
}

// nonProxy handles non-proxy requests
func (p *Proxy) nonProxy(w http.ResponseWriter, req *http.Request) {
	if p.Options.Auth != "" {
		user, password, ok := req.BasicAuth()

		if ok {
			if fmt.Sprintf("%s:%s", user, password) != p.Options.Auth {
				http.Error(w, "Invalid proxy authorization", 407)
				return
			}
		} else {
			http.Error(w, "Invalid proxy authorization", 407)
			return
		}
	}

	if common.Version != "" {
		w.Header().Add("X-Mubeng-Version", common.Version)
	}

	switch req.URL.Path {
	case "/cert":
		p.certHandler(w, req)

		return
	case "/list":
		p.proxyListHandler(w, req)

		return
	}

	http.Error(w, "This is a mubeng proxy server. Does not respond to non-proxy requests.", 500)
}

func (p *Proxy) certHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprint("attachment; filename=", "goproxy-cacert.der"))
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(goproxy.GoproxyCa.Certificate[0]); err != nil {
		http.Error(w, "Failed to get proxy certificate authority.", 500)
		log.Errorf("%s %s %s %s", req.RemoteAddr, req.Method, req.URL, err.Error())
		return
	}

	return
}

func (p *Proxy) proxyListHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	countryQuery := req.URL.Query().Get("country")
	cityQuery := req.URL.Query().Get("city")

	file, err := os.Open(p.Options.File)

	if err != nil {
		http.Error(w, "Failed to get proxies list", 500)
		log.Errorf("%s %s %s %s", req.RemoteAddr, req.Method, req.URL, err.Error())
		return
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	proxies := ProxyInfoList{}
	proxies.Proxies = make([]ProxyInfo, 0)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Split(line, "|")

		if len(parts) != 3 {
			log.Errorf("%s %s %s %s", req.RemoteAddr, req.Method, req.URL, fmt.Sprintf("Invalid proxy %s", line))
			continue
		}

		city := parts[2]

		if cityQuery != "" && city != cityQuery {
			continue
		}

		country := parts[1]

		if countryQuery != "" && country != countryQuery {
			continue
		}

		proxy := parts[0]

		proxyInfo := ProxyInfo{
			Address: proxy,
			Country: country,
			City:    city,
		}

		proxies.Proxies = append(proxies.Proxies, proxyInfo)
	}

	proxiesResponse, err := json.Marshal(proxies)

	if err != nil {
		http.Error(w, "Failed to get proxies list", 500)
		log.Errorf("%s %s %s %s", req.RemoteAddr, req.Method, req.URL, err.Error())
		return
	}

	if _, err := w.Write(proxiesResponse); err != nil {
		http.Error(w, "Failed to get proxies list", 500)
		log.Errorf("%s %s %s %s", req.RemoteAddr, req.Method, req.URL, err.Error())
		return
	}

	return
}
