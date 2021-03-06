package main

import (
	"cache"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	cache.NewGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:9999" //use the localhost now
	peers := cache.NewHTTPPool(addr)
	log.Println("cache is running at", addr)
	//ListenAndServe(a,b)  b needs a func serveHTTP()
	log.Fatal(http.ListenAndServe(addr, peers))
}
