package lru

import (
	"container/list"
)

type Cache struct {
	maxBytes  int64                         //maxMem
	nbytes    int64                         //usedMem
	ll        *list.List                    //head-tail list
	Cache     map[string]*list.Element      //key-value
	onEvicted func(key string, value Value) //remove call-back func
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		Cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.Cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() //remove LRU
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)

		delete(c.Cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.Cache[key]; ok {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.Cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	//if room if full ，remove exist ones
	//may remove more than one
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

//the num of elements in cache list
func (c *Cache) Len() int {
	return c.ll.Len()
}
