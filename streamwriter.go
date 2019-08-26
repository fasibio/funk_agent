package main

import "github.com/gorilla/websocket"

type Serverwriter = func(con *websocket.Conn, msg []Message) error

func WriteToServer(con *websocket.Conn, msg []Message) error {
	return con.WriteJSON(msg)
}
