package proxymanager

import (
	"log"
	"math/rand"

	"github.com/fsnotify/fsnotify"
)

// NextProxy will navigate the next proxy to use
func (p *ProxyManager) NextProxy() string {
	p.CurrentIndex++
	if p.CurrentIndex > len(p.Proxies)-1 {
		p.CurrentIndex = 0
	}

	proxy := p.Proxies[p.CurrentIndex]

	return proxy
}

// RandomProxy will choose a proxy randomly from the list
func (p *ProxyManager) RandomProxy() string {
	return p.Proxies[rand.Intn(len(p.Proxies))]
}

// Watch proxy file from events
func (p *ProxyManager) Watch() (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return watcher, err
	}

	if err := watcher.Add(p.Filepath); err != nil {
		return watcher, err
	}

	return watcher, nil
}

// Reload proxy pool
func (p *ProxyManager) Reload() error {
	i := p.CurrentIndex
	diedProxies := p.DiedProxies
	liveProxies := p.LiveProxies

	p, err := New(p.Filepath, p.RotationMethod)

	if err != nil {
		return err
	}

	p.CurrentIndex = i
	p.DiedProxies = diedProxies
	p.LiveProxies = liveProxies

	return nil
}

func (p *ProxyManager) WatchFile(w *fsnotify.Watcher) {
	for {
		select {
		case event := <-w.Events:
			if event.Op == 2 {
				log.Printf("Proxy file has changed, reloading...")

				err := p.Reload()
				if err != nil {
					log.Fatal(err)
				}
			}
		case err := <-w.Errors:
			log.Fatal(err)
		}
	}
}
