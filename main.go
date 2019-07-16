package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fasibio/funk_agent/tracker"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli"
)

type Holder struct {
	streamCon          *websocket.Conn
	Props              Props
	itSelfNamedHost    string
	client             *client.Client
	trackingContainers map[string]*tracker.Tracker
}

type Props struct {
	FunkServerUrl      string
	InsecureSkipVerify bool
<<<<<<< HEAD
=======
	TrackAll           bool
	Connectionkey      string
>>>>>>> 3f9d1e57df83fd4e9f512174725cacbfef351561
}

func main() {
	log.Println("Letz go ")
	app := cli.NewApp()
	app.Name = "Funk Agent"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "insecureSkipVerify",
			EnvVar: "INSECURE_SKIP_VERIFY",
			Usage:  "Allow insecure serverconnections",
		},
		cli.StringFlag{
			Name:   "funkserver",
			EnvVar: "FUNK_SERVER",
			Value:  "ws://localhost:3000",
			Usage:  "the url of the funk_server",
		},
		cli.StringFlag{
			Name:   "connectionkey",
			EnvVar: "CONNECTION_KEY",
			Value:  "changeMe04cf242924f6b5f96",
			Usage:  "The connectionkey given to the funk-server to connect",
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	holder := Holder{
		Props: Props{
			FunkServerUrl:      c.String("funkserver"),
			InsecureSkipVerify: c.Bool("insecureSkipVerify"),
<<<<<<< HEAD
=======
			TrackAll:           c.Bool("trackall"),
			Connectionkey:      c.String("connectionkey"),
>>>>>>> 3f9d1e57df83fd4e9f512174725cacbfef351561
		},
		itSelfNamedHost:    "localhost",
		trackingContainers: make(map[string]*tracker.Tracker),
	}
	err := holder.openSocketConn()
	if err != nil {
		return err
	}

	containerChan := make(chan []types.Container, 1)
	cli, err := StartListeningForContainer(context.Background(), containerChan)
	if err != nil {
		panic(err)
	}

	mu := sync.Mutex{}

	holder.client = cli
	go func() {
		for {
			for c := range containerChan {
				mu.Lock()
				for _, v := range c {
					d, exist := holder.trackingContainers[v.ID]
					if exist {
						d.Container = v
					} else {
						holder.trackingContainers[v.ID] = tracker.NewTracker(holder.client, v)
						// log.Println(v.ID, w.trackingContainers[v.ID])
					}
				}
				mu.Unlock()
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	for {
		for range ticker.C {
			mu.Lock()
			holder.SaveTrackingInfo()
			mu.Unlock()
		}
	}

}

func (w *Holder) SaveTrackingInfo() {
	var msg []Message
	for _, v := range w.trackingContainers {
		logs := w.getLogs(v)
		if logs != nil {
			msg = append(msg, *logs)
		}
		stats := w.getStatsInfo(v)
		if stats != nil {
			msg = append(msg, *stats)
		}
	}
	WriteToServer(w.streamCon, msg)
}

func (w *Holder) getStatsInfo(v *tracker.Tracker) *Message {
	if v.Container.Labels["funk.log.stats"] == "false" {
		log.Println("No stats Logging for ", v.Container.Image)
		return nil
	}
	stats := v.GetStats()

	b, err := json.Marshal(stats)
	if err != nil {
		log.Println(err)
		return nil
	}

	return &Message{
		Time:          time.Now(),
		Type:          MessageType_Stats,
		Data:          []string{string(b)},
		Containername: v.Container.Image,
		Host:          w.itSelfNamedHost,
		SearchIndex:   v.SearchIndex() + "_stats",
		ContainerID:   v.Container.ImageID,
	}
}

func (w *Holder) getLogs(v *tracker.Tracker) *Message {
	if v.Container.Labels["funk.log.logs"] == "false" {
		log.Println("No logs Logging for ", v.Container.Image)
		return nil
	}
	logs := v.GetLogs()
	var strLogs []string

	for _, value := range logs {
		strLogs = append(strLogs, string(value))

	}

	if len(strLogs) > 0 {
		log.Println("Values !!! ")
		return &Message{
			Time:          time.Now(),
			Type:          MessageType_Log,
			Data:          strLogs,
			Containername: v.Container.Image,
			Host:          w.itSelfNamedHost,
			SearchIndex:   v.SearchIndex() + "_logs",
			ContainerID:   v.Container.ImageID,
		}

	} else {
		log.Println("no vlaues")
		return nil
	}
}

func openSocketConnection(url string, isDone *bool, h *Holder, isConnOpen *bool, connectionString string) (*websocket.Conn, error) {
	d := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpHeader := make(http.Header)
	httpHeader.Add("funk.connection", connectionString)

	c, _, err := d.Dial(url, httpHeader)
	if err != nil {
		return nil, err
	}
	return c, nil

}

func (h *Holder) openSocketConn() error {
	if h.streamCon == nil {
		done := false
		conn := true
		d, err := openSocketConnection(h.Props.FunkServerUrl+"/data/subscribe", &done, h, &conn, h.Props.Connectionkey)
		if err != nil {
			return err
		}
		h.streamCon = d
		// go h.handleInterrupt(&done)
		// go h.checkConnAndPoll(&conn, &done)
	}
	return nil
}
