package tccache

import "container/list"

type Entry[K comparable, V any] struct {
	k K
	v V
}

type LRUCache[K comparable, V any] struct {
	capacity int
	cache    map[K]*list.Element
	ll       *list.List

	onDelete func(key K)
}

func New[K comparable, V any](capacity int, onDelete func(key K)) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		cache:    make(map[K]*list.Element),
		ll:       list.New(),
		onDelete: onDelete,
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if ent, exists := c.cache[key]; exists {
		c.ll.MoveToFront(ent)
		return ent.Value.(*Entry[K, V]).v, true
	}
	return value, false
}

func (c *LRUCache[K, V]) Put(key K, value V) {
	if ent, exists := c.cache[key]; exists {
		c.ll.MoveToFront(ent)
		ent.Value.(*Entry[K, V]).v = value
		return
	}

	ent := &Entry[K, V]{k: key, v: value}
	c.cache[key] = c.ll.PushFront(ent)

	if c.ll.Len() > c.capacity {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.cache, last.Value.(*Entry[K, V]).k)
		if c.onDelete != nil {
			c.onDelete(last.Value.(*Entry[K, V]).k)
		}
	}
}

func (c *LRUCache[K, V]) Delete(key K) {
	if ent, exists := c.cache[key]; exists {
		c.ll.Remove(ent)
		delete(c.cache, key)
		if c.onDelete != nil {
			c.onDelete(key)
		}
	}
}

func (c *LRUCache[K, V]) Clear() {
	for k := range c.cache {
		if c.onDelete != nil {
			c.onDelete(k)
		}
	}
	c.cache = make(map[K]*list.Element)
	c.ll.Init()
}
