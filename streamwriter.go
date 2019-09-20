package main

import "github.com/gorilla/websocket"

// Serverwriter is the definition how a message have to look like to send them to server
type Serverwriter = func(con *websocket.Conn, msg []Message) error

// WriteToServer is the implementation of Serverwriter
func WriteToServer(con *websocket.Conn, msg []Message) error {
	return con.WriteJSON(msg)
}
