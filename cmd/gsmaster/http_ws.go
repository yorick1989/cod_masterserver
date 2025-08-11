package main

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type WSServer struct {
	conns map[*websocket.Conn]struct{}
	mu    sync.Mutex
}

func NewServer() *WSServer {
	return &WSServer{
		conns: make(map[*websocket.Conn]struct{}),
	}
}

func (s *WSServer) handleWS(ws *websocket.Conn) {
	fmt.Println("New incoming connection:", ws.RemoteAddr().String())

	s.mu.Lock()
	s.conns[ws] = struct{}{}
	s.mu.Unlock()

	s.readLoop(ws)
}

func (s *WSServer) writeLoop() {

	for {
		timeStamp := time.Now().Format("2006-01-02 15:04:05")
		for ws, _ := range s.conns {

			fmt.Println("YORICKKK2", timeStamp)
			html := `
			<div hx-swap-oob="innerHTML:#update-timestamp">` + timeStamp + `</div>
			`
			ws.Write([]byte(html))
		}
		time.Sleep(time.Second * 1)
	}
}

func (s *WSServer) readLoop(ws *websocket.Conn) {
	fmt.Println("New incoming connection", ws.RemoteAddr().String())

	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error:", err)
			continue
		}
		msg := buf[:n]
		fmt.Println(string(msg))
		ws.Write([]byte("Thank you for the message:" + string(msg)))

	}

	s.mu.Lock()
	delete(s.conns, ws)
	s.mu.Unlock()

	fmt.Println("Closed connection connection", ws.RemoteAddr().String())
}
