# CacheDesign


> Go实现一个分布式的缓存器

## Cache-Basic LRU

1.cache 主要结构是双向链表，固定内存
```
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}
```

2.使用go 的testing包进行测试：所有的函数写成Test开头的，报错的地方使用t.Fatalf(errrorString),go test即能完成测试

