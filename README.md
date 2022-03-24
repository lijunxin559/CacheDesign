# CacheDesign


> Go实现一个分布式的缓存器
> 
> [教室](https://geektutu.com/post/geecache.html)
<br/></br>
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

<br/></br>
## 2.单机并发缓存

1.首先学习sync.Mutex互斥锁，lock and Unlock

2.添加ByteView只读数据结构，表示缓存的值，可以复制读取，但是不能修改

3.主体结构Group是一个缓存的命名空间，可以用来得到数据
```
流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
流程 ⑶ ：缓存不存在，则调用 load 方法，load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），getLocally 调用用户回调函数
```

<br/></br>
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

<br/></br>
## 4.一致性hash

1.增加了一个Map结构
有多个真实的节点，一个真实的服务节点对应多个虚拟服务节点
虚拟环上的值存储在下一个顺时针的服务节点（虚节点）的实节点

2.Map结构需要实现
Get():得到一个值对应的实节点
Add():增加一个实服务节点的时候设置相应虚拟服务节点的map映射

3.Map中映射的方式可以使用函数接口提供给用户自定义


<br/></br>
## 5.分布式节点

1.两个流程
查找数据流程：
```
                           是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
```

从远端获取数据流程：
```
使用一致性哈希选择节点        是                                    是
    |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
                    |  否                                    ↓  否
                    |----------------------------> 回退到本地节点处理。
```

2.main函数中:
startCacheServer() 用来启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 gee 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
startAPIServer() 用来启动一个 API 服务（端口 9999），与用户进行交互，用户感知
main() 函数需要命令行传入 port 和 api 2 个参数，用来在指定端口启动 HTTP 服务

3.测试可以看到，我们并发了 3 个请求 ?key=Tom，从日志中可以看到，三次均选择了节点 8001，这是一致性哈希算法的功劳，但是发起了3次请求，且返回的是相同的数据

<br/></br>
## 6.防止缓存击穿
>缓存雪崩：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。
缓存击穿：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增。
缓存穿透：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

1.定义call结构为正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入。

2.group 是 singleflight 的主数据结构，管理不同 key 的请求(call)。

3.group的Do 方法，接收 2 个参数，第一个参数是 key，第二个参数是一个函数 fn。Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。

4.在修改 geecache.go 中的 Group，添加成员变量 loader，并更新构建函数 NewGroup
修改 load 函数，将原来的 load 的逻辑，使用 g.loader.Do 包裹起来即可，这样确保了并发场景下针对相同的 key，load 过程只会调用一次。

5.测试会发现，三次对Tom的并发请求只会请求一次，注：在run.sh中，curl使用&能够实现并发,但其实，当并发的数量比较大的时候，for循环并不能赶上查询的速度，所以还是会查询多次。

<em>但并发请求50次时为什么返回的次数并不是请求的次数？</em>

<br/></br>
## 7.protobuf 通信
>[protobuf教程](https://geektutu.com/post/gee-day2.html)
<br/>分为两步：
<br/>1）在 .proto 文件中定义数据结构，并使用 protoc 生成 Go 代码（.proto 文件是跨平台的，还可以生成 C、Java 等其他源码文件。</br>
<br/>2）在项目代码中引用生成的 Go 代码。</br>
</br>

1.新建package pb,定义Request和Response

2.在PeerGetter.Get()->Group.getFromPeer()->HTTPPool.ServeHTTP()\httpGetter.Get()中修改接口调用方式


<br/></br>
## 总结
lru：为了解决资源限制问题

mutex：单机并发，给用户提供了自定义数据源的回调函数

http:实现服务端

consistentHash：实现一致性哈希算法，解决远程节点挑选问题

distributed：创建HTTP客户端，实现多节点之间的通信

singleflight：解决缓存击穿问题

protobuf：优化节点间的通信性能

