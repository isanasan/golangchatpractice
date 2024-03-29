package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type room struct {
    // forward is a channel for send to other client
    forward chan []byte
    // join is a channel for client that join
    join chan *client
    // leave is a channel fo clinet that leave
    leave chan *client
    // clients has all clients of in the room
    clients map[*client]bool
}

// newroom is make chatroom quickly
func newRoom() *room {
    return &room{
        forward: make(chan []byte),
        join: make(chan *client),
        leave: make(chan *client),
        clients: make(map[*client]bool),
    }
}

func (r *room) run() {
    for {
        select {
        case client := <-r.join:
            //join
            r.clients[client] = true
        case client := <-r.leave:
            //leave
            delete(r.clients,client)
            close(client.send)
        case msg := <-r.forward:
            //send message to all client
            for client := range r.clients {
                select {
                case client.send <- msg:
                    // send message
                default:
                    // faild to send message
                    delete(r.clients,client)
                    close(client.send)
                }
            }
        }
    }
}

const (
    socketBufferSize = 1024
    messageBufferSize = 256
)
var upgrader = &websocket.Upgrader{ReadBufferSize:socketBufferSize,WriteBufferSize: socketBufferSize}
func (r *room) ServeHTTP(w http.ResponseWriter,req *http.Request) {
    upgrader.CheckOrigin = func(r *http.Request) bool { return true }
    socket,err := upgrader.Upgrade(w,req,nil)
    if err != nil {
        log.Fatal("ServeHTTP:",err)
        return
    }
    client := &client{
        socket: socket,
        send:make(chan []byte,messageBufferSize),
        room:r,
    }
    r.join <- client
    defer func() { r.leave <- client  }()
    go client.write()
    client.read()
}
