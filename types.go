package main

import "time"

type MessageType string

const (
	MessageType_Log   MessageType = "LOG"
	MessageType_Stats MessageType = "STATS"
)

type Message struct {
	Time          time.Time   `json:"time,omitempty"`
	Type          MessageType `json:"type,omitempty"`
	Data          []string    `json:"data,omitempty"`
	Containername string      `json:"containername,omitempty"`
	Servicename   string      `json:"servicename,omitempty"`
	Namespace     string      `json:"namespace,omitempty"`
	ContainerID   string      `json:"container_id,omitempty"`
	Host          string      `json:"host,omitempty"`
	SearchIndex   string      `json:"search_index,omitempty"`
}
