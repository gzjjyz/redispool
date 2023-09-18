package redispool

import (
	"container/list"
	"fmt"
	"github.com/gzjjyz/logger"
	"gopkg.in/redis.v5"
	"sync"
	"time"
)

const (
	DefaultPort = 6379
	DefaultMin  = 8
)

type Pool struct {
	host       string
	password   string
	port       int
	db         int
	totalCount int
	minCount   int
	clients    *list.List
	changeAt   time.Time
	mutex      sync.Mutex
}

func NewPool(opts ...Option) (*Pool, error) {
	pool := &Pool{}

	for _, opt := range opts {
		opt(pool)
	}

	if pool.port == 0 {
		pool.port = DefaultPort
		logger.Info("redis port set default value(%d)", DefaultPort)
	}

	if pool.minCount <= 0 {
		pool.minCount = DefaultMin
		logger.Info("redis client count set default value(%d)", DefaultMin)
	}

	pool.clients = list.New()

	client := pool.Pop()
	defer pool.Push(client)

	if err := client.Ping().Err(); err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *Pool) Pop() *redis.Client {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.clients.Len() == 0 {
		host := fmt.Sprintf("%s:%d", p.host, p.port)
		for i := 0; i < p.minCount; i++ {
			p.clients.PushBack(redis.NewClient(&redis.Options{
				Addr:     host,
				Password: p.password,
				DB:       p.db,
			}))
		}
		p.changeAt = time.Now()
		p.totalCount += p.minCount
	}
	client := p.clients.Front().Value.(*redis.Client)
	p.clients.Remove(p.clients.Front())
	return client
}

func (p *Pool) Push(client *redis.Client) {
	if nil == client {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.clients.PushBack(client)

	p.gc()
}

func (p *Pool) gc() {
	dur := time.Since(p.changeAt)
	if dur <= time.Minute {
		return
	}

	count := p.totalCount / 10

	if dur >= time.Hour {
		count = p.totalCount - p.minCount
	} else if dur >= time.Minute*30 {
		count = p.totalCount * 8 / 10
	} else if dur >= time.Minute*10 {
		count = p.totalCount * 3 / 10
	}

	if count <= 0 {
		return
	}

	if p.clients.Len() <= p.minCount {
		return
	}

	for i := 0; i < count && p.clients.Len() > 1; i++ {
		p.totalCount -= 1
		client := p.clients.Front().Value.(*redis.Client)
		p.clients.Remove(p.clients.Front())
		err := client.Close()
		if err != nil {
			logger.Errorf("free redis client error! %v", err)
		}
	}

	logger.Warn("redis clear:free-connect-cnt: %d, total-connect-cnt: %d", p.clients.Len(), p.totalCount)
	p.changeAt = time.Now()
}
