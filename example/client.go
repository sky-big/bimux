package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"

	".."
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/mux"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("dial:", err)
	}
	defer c.Close()

	mux, _ := bimux.NewWebSocketMuxer(c,
		func(route uint32, req []byte) []byte {
			switch route {
			case 1:
				req = append(req, []byte("ok")...)
				return req
			}
			return nil
		},
		nil)
	mux.Wait()
	fmt.Println("client over")

}
