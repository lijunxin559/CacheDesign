# CacheDesign


> Go实现一个分布式的缓存器

## 1.Cache-Basic LRU

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

## 2.单机并发缓存

1.首先学习sync.Mutex互斥锁，lock and Unlock

2.添加ByteView只读数据结构，表示缓存的值，可以复制读取，但是不能修改

3.主体结构Group是一个缓存的命名空间，可以用来得到数据
```
流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
流程 ⑶ ：缓存不存在，则调用 load 方法，load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），getLocally 调用用户回调函数
```

## 3.http服务端

想要完成的结构如下
```
geecache/
    |--lru/
        |--lru.go  // lru 缓存淘汰策略
    |--byteview.go // 缓存值的抽象与封装
    |--cache.go    // 并发控制
    |--geecache.go // 负责与外部交互，控制缓存存储和获取的主流程
	|--http.go     // 提供被其他节点访问的能力(基于http)
```


1.HTTPPool作为承载节点间HTTP的通信核心数据结构：
self，用来记录自己的地址，包括主机名/IP 和端口
basePath，作为节点间通讯地址的前缀，默认是/_cache/
（note：因为一个主机上还可能承载其他的服务，加一个Path是一个好习惯，大部分的API接口一般以/api作为前缀）

2.其中HTTPPool的ServeHTTP实现：
我们约定访问路径格式为 /<basepath>/<groupname>/<key>
首先判断路径前缀是不是basePath，不是的话直接返回错误
否则通过路径直接得到group实例，使用group.Get(key)得到数据
将key作为httpResponse的body写入返回

3.暂时只在本地建立数据库并进行测试
