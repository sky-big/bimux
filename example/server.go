package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "net/http/pprof"

	".."
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func handler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade:", err)
		return
	}
	defer c.Close()

	muxer, _ := bimux.NewWebSocketMuxer(c,
		func(route uint32, req []byte) []byte {
			fmt.Println("err rpc")
			return nil
		},
		func(route uint32, req []byte) {
			fmt.Println("err oneway receive")
		})
	var wg sync.WaitGroup
	for x := 0; x < 10; x++ {
		wg.Add(1)
		go func(y int) {
			for i := 0; i < 10000; i++ {
				req := fmt.Sprintf("%v %v", y, i)
				rsp, err := muxer.Rpc(1, []byte(req), time.Second)
				if err != nil || string(rsp[:]) != req+"ok" {
					fmt.Printf("rpc return err %v[req:%s, rsp:%s]\n", err, req, rsp)
				}
			}
			wg.Done()
		}(x)
	}
	wg.Wait()
	fmt.Println("conn over ")
}

func main() {
	flag.Parse()
	http.HandleFunc("/mux", handler)
	fmt.Println(http.ListenAndServe(*addr, nil))
}
