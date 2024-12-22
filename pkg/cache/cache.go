package cache

import (
  "sync"
)

type Key[Primary, Secondary comparable] struct {
  P Primary
  S Secondary
}

type Cache[Primary, Secondary comparable, Value any] struct {
  mu     sync.Mutex
  values map[Primary]map[Secondary]Value
}

func NewCache[P, S comparable, V any]() *Cache[P, S, V] {
  return &Cache[P, S, V]{
    values: make(map[P]map[S]V),
  }
}

func (c *Cache[P, S, V]) Set(key Key[P, S], value V) {
  c.mu.Lock()
  defer c.mu.Unlock()

  if c.values[key.P] == nil {
    c.values[key.P] = make(map[S]V)
  }

  c.values[key.P][key.S] = value
}

func (c *Cache[P, S, V]) Get(key Key[P, S]) (value V, ok bool) {
  c.mu.Lock()
  defer c.mu.Unlock()

  if _, ok = c.values[key.P]; !ok {
    return value, false
  }

  value, ok = c.values[key.P][key.S]

  return value, ok
}

func (c *Cache[P, S, V]) DeleteP(key P) {
  c.mu.Lock()
  defer c.mu.Unlock()

  delete(c.values, key)
}

func (c *Cache[P, S, V]) DeleteS(key Key[P, S]) {
  c.mu.Lock()
  defer c.mu.Unlock()

  if _, ok := c.values[key.P]; !ok {
    return
  }

  delete(c.values[key.P], key.S)
}

func (c *Cache[P, S, V]) Clear() {
  c.mu.Lock()
  defer c.mu.Unlock()

  clear(c.values)
}
