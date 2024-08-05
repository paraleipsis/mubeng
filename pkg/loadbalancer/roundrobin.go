package loadbalancer

import (
	"slices"
	"sync"
)

type LoadBalancer[T comparable] struct {
	items       []T
	mu          sync.Mutex
	itemChannel chan T
	commitChan  chan T
	onCommitFn  *func()
}

func NewLoadBalancer[T comparable](
	onCommitFn *func(),
) *LoadBalancer[T] {
	return &LoadBalancer[T]{
		onCommitFn: onCommitFn,
		commitChan: make(chan T),
	}
}

func (l *LoadBalancer[T]) Next() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case item := <-l.itemChannel:
		return item, true
	default:
		return *new(T), false
	}
}

func (l *LoadBalancer[T]) Commit(item T) {
	if slices.Contains(l.items, item) {
		l.commitChan <- item
	}
}

func (l *LoadBalancer[T]) runUnlocker() {
	for {
		item := <-l.commitChan

		go func(item T) {
			if l.onCommitFn != nil {
				fn := *l.onCommitFn
				fn()
			}

			l.itemChannel <- item
		}(item)
	}
}

func (l *LoadBalancer[T]) AddItems(items ...T) {
	l.items = append(l.items, items...)

	l.itemChannel = make(chan T, len(items))

	for _, item := range items {
		l.itemChannel <- item
	}

	go l.runUnlocker()
}
