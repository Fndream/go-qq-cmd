package cache

import (
	"sync"
	"time"
)

type Cache struct {
	cache         *sync.Map
	defaultExpire time.Duration
}

type CValue struct {
	value      interface{}
	expireTime time.Time
}

func New(expire time.Duration, cleanCycle time.Duration) *Cache {
	cache := &Cache{cache: &sync.Map{}, defaultExpire: expire}
	if cleanCycle > 0 {
		cache.startCleanTimer(cleanCycle)
	}
	return cache
}

func (c *Cache) Get(key string) (interface{}, bool) {
	v, ok := c.cache.Load(key)
	if !ok {
		return nil, false
	}

	val := v.(*CValue)
	if c.IsExpired(val) {
		c.Delete(key)
		return nil, false
	}
	return val.value, true
}

func (c *Cache) Set(key string, value interface{}) {
	c.cache.Store(key, &CValue{value: value, expireTime: time.Now().Add(c.defaultExpire)})
}

func (c *Cache) SetExpire(key string, value interface{}, expire time.Duration) {
	c.cache.Store(key, &CValue{value: value, expireTime: time.Now().Add(expire)})
}

func (c *Cache) Delete(key string) {
	c.cache.Delete(key)
}

func (c *Cache) IsExpired(val *CValue) bool {
	return !val.expireTime.IsZero() && time.Now().After(val.expireTime)
}

func (c *Cache) cleanExpired() int {
	var delKeys []string
	c.cache.Range(func(key, value interface{}) bool {
		val := value.(*CValue)
		if c.IsExpired(val) {
			delKeys = append(delKeys, key.(string))
		}
		return true
	})
	for _, key := range delKeys {
		c.cache.Delete(key)
	}
	return len(delKeys)
}

func (c *Cache) startCleanTimer(cycle time.Duration) {
	if cycle < time.Minute {
		cycle = time.Minute // 最小清理周期为1分钟
	}
	go func() {
		ticker := time.NewTicker(cycle)
		for {
			select {
			case <-ticker.C:
				c.cleanExpired()
			}
		}
	}()
}
