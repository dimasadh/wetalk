package cache

import (
	"errors"
	"sync"
	"time"
)

// MemCache is a simple in-memory cache backed by sync.Map.
// Items can have optional TTL. A background cleanup goroutine
// runs when NewMemCache is given a positive cleanupInterval.
type MemCache struct {
	items sync.Map
	stop  chan struct{}
	wg    sync.WaitGroup
}

type item struct {
	mu         sync.Mutex
	value      any
	expiration int64 // unix nano; 0 means no expiration
}

// NewMemCache creates a new MemCache. If cleanupInterval > 0,
// a background goroutine will periodically remove expired items.
func NewMemCache(cleanupInterval time.Duration) *MemCache {
	m := &MemCache{
		stop: make(chan struct{}),
	}
	if cleanupInterval > 0 {
		m.wg.Add(1)
		go func() {
			ticker := time.NewTicker(cleanupInterval)
			defer ticker.Stop()
			defer m.wg.Done()
			for {
				select {
				case <-ticker.C:
					m.cleanup()
				case <-m.stop:
					return
				}
			}
		}()
	}
	return m
}

func (m *MemCache) Set(key string, value any, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	m.items.Store(key, &item{
		value:      value,
		expiration: exp,
	})
}

func (m *MemCache) Get(key string) (any, bool) {
	v, ok := m.items.Load(key)
	if !ok {
		return nil, false
	}
	it := v.(*item)
	if it.isExpired() {
		m.items.Delete(key)
		return nil, false
	}
	return it.value, true
}

func (m *MemCache) Delete(key string) {
	m.items.Delete(key)
}

func (m *MemCache) Exists(key string) bool {
	_, ok := m.Get(key)
	return ok
}

func (m *MemCache) Flush() {
	m.items.Range(func(k, _ any) bool {
		m.items.Delete(k)
		return true
	})
}

func (m *MemCache) Close() {
	if m.stop == nil {
		return
	}
	close(m.stop)
	m.wg.Wait()
}

func (m *MemCache) Keys() []string {
	keys := make([]string, 0)
	now := time.Now().UnixNano()
	m.items.Range(func(k, v any) bool {
		it := v.(*item)
		if it.expiration == 0 || now <= it.expiration {
			if ks, ok := k.(string); ok {
				keys = append(keys, ks)
			}
		}
		return true
	})
	return keys
}

func (m *MemCache) Range(f func(key, value any) bool) {
	now := time.Now().UnixNano()
	m.items.Range(func(k, v any) bool {
		it := v.(*item)
		if it.expiration == 0 || now <= it.expiration {
			return f(k, it.value)
		}
		return true
	})
}

// Increment increases an integer value stored at key by delta.
// If the key does not exist it will be created with value delta.
// Returns the new value or an error if the existing value is not an integer.
var ErrNotInteger = errors.New("value is not an integer")

func (m *MemCache) Increment(key string, delta int64) (int64, error) {
	// Ensure an item exists for the key.
	actual, _ := m.items.LoadOrStore(key, &item{
		value:      int64(0),
		expiration: 0,
	})
	it := actual.(*item)

	it.mu.Lock()
	defer it.mu.Unlock()

	if it.isExpired() {
		// treat as not present: reset
		it.value = int64(0)
		it.expiration = 0
	}

	switch v := it.value.(type) {
	case int:
		newv := int64(v) + delta
		it.value = int(newv)
		return newv, nil
	case int32:
		newv := int64(v) + delta
		it.value = int32(newv)
		return newv, nil
	case int64:
		newv := v + delta
		it.value = newv
		return newv, nil
	case uint:
		newv := int64(v) + delta
		it.value = uint(newv)
		return newv, nil
	case uint32:
		newv := int64(v) + delta
		it.value = uint32(newv)
		return newv, nil
	case uint64:
		newv := int64(v) + delta
		it.value = uint64(newv)
		return newv, nil
	default:
		// if it's zero-value type (e.g. nil) replace with delta as int64
		if it.value == nil {
			it.value = delta
			return delta, nil
		}
		return 0, ErrNotInteger
	}
}

func (m *MemCache) Decrement(key string, delta int64) (int64, error) {
	return m.Increment(key, -delta)
}

func (it *item) isExpired() bool {
	if it == nil || it.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > it.expiration
}

func (m *MemCache) cleanup() {
	now := time.Now().UnixNano()
	m.items.Range(func(k, v any) bool {
		it := v.(*item)
		if it.expiration != 0 && now > it.expiration {
			m.items.Delete(k)
		}
		return true
	})
}
