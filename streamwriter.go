package main

import "github.com/gorilla/websocket"

func WriteToServer(con *websocket.Conn, msg []Message) error {
	return con.WriteJSON(msg)
}
