package cmd

import (
	"sync"
	"time"
)

var mu = sync.Mutex{}
var locks = map[string]*sync.Mutex{}
var activity = map[string]*time.Time{}

func init() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				for k, v := range locks {
					t := activity[k]
					if time.Since(*t) > time.Minute*5 {
						if ok := v.TryLock(); ok {
							delete(activity, k)
							delete(locks, k)
							v.Unlock()
						}
					}
				}
				mu.Unlock()
			}
		}
	}()
}

func lock(key string) *sync.Mutex {
	if lock, ok := locks[key]; ok {
		lock.Lock()
		if lock, ok = locks[key]; ok {
			now := time.Now()
			activity[key] = &now
			return lock
		}
		lock.Unlock()
	}

	mu.Lock()
	defer mu.Unlock()
	if lock, ok := locks[key]; ok {
		now := time.Now()
		activity[key] = &now
		return lock
	}
	m := &sync.Mutex{}
	m.Lock()
	locks[key] = m
	now := time.Now()
	activity[key] = &now
	return m
}

func unLock(lock *sync.Mutex) {
	lock.Unlock()
}
