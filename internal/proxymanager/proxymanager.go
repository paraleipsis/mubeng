package proxymanager

import (
	"bufio"
	"fmt"
	"ktbs.dev/mubeng/pkg/helper"
	"ktbs.dev/mubeng/pkg/loadbalancer"
	"ktbs.dev/mubeng/pkg/mubeng"
	"math/rand"
	"os"
	"strings"
	"time"
)

// ProxyManager defines the proxy list and current proxy position
type ProxyManager struct {
	CurrentIndex   int
	Filepath       string
	Length         int
	Proxies        []string
	DiedProxies    []string
	LiveProxies    []string
	RoundRobin     *loadbalancer.LoadBalancer[string]
	RotationMethod string
}

func init() {
	rand.Seed(time.Now().UnixNano())

	manager = &ProxyManager{CurrentIndex: -1}
}

// New initialize ProxyManager
func New(filename string, rotationMethod string) (*ProxyManager, error) {
	keys := make(map[string]bool)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	manager.Proxies = []string{}
	manager.Filepath = filename
	manager.RotationMethod = rotationMethod

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := helper.Eval(scanner.Text())
		proxy = strings.Split(proxy, "|")[0]

		if _, value := keys[proxy]; !value {
			_, err = mubeng.Transport(placeholder.ReplaceAllString(proxy, ""))
			if err == nil {
				keys[proxy] = true
				manager.Proxies = append(manager.Proxies, proxy)
			}
		}
	}

	manager.Length = len(manager.Proxies)
	if manager.Length < 1 {
		return manager, fmt.Errorf("open %s: has no valid proxy URLs", filename)
	}

	if rotationMethod == "round-robin" {
		rateLimiter := func() {
			time.Sleep(0 * time.Millisecond)
		}

		rr := loadbalancer.NewLoadBalancer[string](&rateLimiter)

		manager.RoundRobin = rr

		manager.RoundRobin.AddItems(manager.Proxies...)
	}

	return manager, scanner.Err()
}
