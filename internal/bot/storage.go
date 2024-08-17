package bot

import (
	"bufio"
	"bytes"
	"context"
	"github.com/projectdiscovery/gologger"
	"ktbs.dev/mubeng/internal/proxymanager"
	"os"
	"slices"
)

type ProxyStatus string

const (
	Online  ProxyStatus = "online"
	Offline ProxyStatus = "offline"
)

type Protocol string

const (
	HTTP  Protocol = "http"
	HTTPS Protocol = "https"
)

type ProxyStorage struct {
	ProxyManager *proxymanager.ProxyManager
}

func NewProxyStorage(proxyManager *proxymanager.ProxyManager) *ProxyStorage {
	return &ProxyStorage{ProxyManager: proxyManager}
}

func (s *ProxyStorage) GetOnlineProxies(_ context.Context) ([]string, error) {
	return s.ProxyManager.LiveProxies, nil
}

func (s *ProxyStorage) GetOfflineProxies(_ context.Context) ([]string, error) {
	return s.ProxyManager.DiedProxies, nil
}

func (s *ProxyStorage) AddProxies(_ context.Context, proxies ...string) error {
	f, err := os.OpenFile(s.ProxyManager.Filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		gologger.Error().Msgf("Error! %s", err)
		return err
	}

	defer func(f *os.File) {
		err = f.Close()

		if err != nil {
			gologger.Error().Msgf("Error! %s", err)
		}
	}(f)

	var result string

	for _, proxy := range proxies {
		result += proxy + "\n"
	}

	if _, err = f.WriteString(result); err != nil {
		gologger.Error().Msgf("Error! %s", err)
		return err
	}

	return nil
}

func (s *ProxyStorage) DeleteProxies(_ context.Context, offline bool, proxies ...string) error {
	f, err := os.Open(s.ProxyManager.Filepath)

	if offline {
		for i, v := range s.ProxyManager.DiedProxies {
			if slices.Contains(proxies, v) {
				s.ProxyManager.DiedProxies = append(s.ProxyManager.DiedProxies[:i], s.ProxyManager.DiedProxies[i+1:]...)
				break
			}
		}
	} else {
		for i, v := range s.ProxyManager.LiveProxies {
			if slices.Contains(proxies, v) {
				s.ProxyManager.LiveProxies = append(s.ProxyManager.LiveProxies[:i], s.ProxyManager.LiveProxies[i+1:]...)
				break
			}
		}
	}

	if err != nil {
		gologger.Error().Msgf("Error! %s", err)
		return err
	}

	defer func(f *os.File) {
		err = f.Close()

		if err != nil {
			gologger.Error().Msgf("Error! %s", err)
		}
	}(f)

	var bs []byte
	buf := bytes.NewBuffer(bs)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if !slices.Contains(proxies, scanner.Text()) {
			_, err := buf.Write(scanner.Bytes())

			if err != nil {
				gologger.Error().Msgf("Error! %s", err)
				return err
			}

			_, err = buf.WriteString("\n")

			if err != nil {
				gologger.Error().Msgf("Error! %s", err)
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		gologger.Error().Msgf("Error! %s", err)
		return err
	}

	err = os.WriteFile(s.ProxyManager.Filepath, buf.Bytes(), 0644)

	if err != nil {
		gologger.Error().Msgf("Error! %s", err)
		return err
	}

	return nil
}
