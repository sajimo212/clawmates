package matching

import (
	"strconv"
	"sync"
)

type pairScore struct {
	score float64
	topic string
}

// PairScoreCache is a pluggable KV store for pair scoring results.
// You can provide a distributed implementation (e.g. Redis) via SetPairScoreCache.
type PairScoreCache interface {
	Get(key string) (pairScore, bool)
	Set(key string, val pairScore)
	GetParam(key, param string) (string, bool)
}

type localPairScoreCache struct {
	mu    sync.RWMutex
	items map[string]pairScore
}

func newLocalPairScoreCache() *localPairScoreCache {
	return &localPairScoreCache{
		items: make(map[string]pairScore, 4096),
	}
}

func (c *localPairScoreCache) Get(key string) (pairScore, bool) {
	c.mu.RLock()
	v, ok := c.items[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *localPairScoreCache) Set(key string, val pairScore) {
	c.mu.Lock()
	c.items[key] = val
	c.mu.Unlock()
}

func (c *localPairScoreCache) GetParam(key, param string) (string, bool) {
	c.mu.RLock()
	v, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}

	switch param {
	case "score":
		return strconv.FormatFloat(v.score, 'f', -1, 64), true
	case "topic":
		return v.topic, true
	default:
		return "", false
	}
}

var scoreCache PairScoreCache = newLocalPairScoreCache()

func SetPairScoreCache(cache PairScoreCache) {
	if cache == nil {
		scoreCache = newLocalPairScoreCache()
		return
	}
	scoreCache = cache
}

// GetCachedPairParam allows callers to fetch a specific cached field in O(1).
// Supported params: "score", "topic".
func GetCachedPairParam(key, param string) (string, bool) {
	return scoreCache.GetParam(key, param)
}
