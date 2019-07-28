package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fasibio/funk_agent/logger"
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

type StatsLog string

// isValidate Check current value is a valid value
func (s StatsLog) isValidate() bool {
	switch s {
	case StatsLogAll:
		return true
	case StatsLogCumulated:
		return true
	case StatsLogNo:
		return true

	}
	return false
}

const (
	StatsLogAll       StatsLog = "all"
	StatsLogCumulated StatsLog = "cumulated"
	StatsLogNo        StatsLog = "no"
)

type Props struct {
	FunkServerUrl      string
	InsecureSkipVerify bool
	Connectionkey      string
	LogStats           StatsLog
}

func main() {
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
		cli.StringFlag{
			Name:   "logstats",
			EnvVar: "LOG_STATS",
			Value:  "all",
			Usage:  "Log the statsinfo three values allowed all, cumulated (not supported now), no",
		},
	}
	if err := app.Run(os.Args); err != nil {
		logger.Get().Fatalw("Global error: " + err.Error())
	}
}

func run(c *cli.Context) error {
	logger.Initialize("info")
	statslog := StatsLog(c.String("logstats"))
	if !statslog.isValidate() {
		return fmt.Errorf("logstats has no valid Parameter" + c.String("logstats"))
	}

	holder := Holder{
		Props: Props{
			FunkServerUrl:      c.String("funkserver"),
			InsecureSkipVerify: c.Bool("insecureSkipVerify"),
			Connectionkey:      c.String("connectionkey"),
			LogStats:           statslog,
		},
		itSelfNamedHost:    "localhost",
		trackingContainers: make(map[string]*tracker.Tracker),
	}
	err := holder.openSocketConn(false)
	for err != nil {
		err = holder.openSocketConn(false)
		logger.Get().Errorw("No connection to Server... Wait 5s and try again later")
		time.Sleep(5 * time.Second)
	}

	logger.Get().Infow("Connected to Funk-Server")
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
	for _, v := range w.trackingContainers {
		var msg []Message
		logs := w.getLogs(v)
		if logs != nil {
			msg = append(msg, *logs)
		}
		if w.Props.LogStats == StatsLogAll {
			stats := w.getStatsInfo(v)
			if stats != nil {
				msg = append(msg, *stats)
			}
		}
		if len(msg) != 0 {
			err := WriteToServer(w.streamCon, msg)
			if err != nil {
				logger.Get().Warnw("Error by write Data to Server" + err.Error() + " try to reconnect")

				err := w.openSocketConn(true)
				if err != nil {
					logger.Get().Warnw("Can not connect try again later: " + err.Error())
				} else {
					logger.Get().Infow("Connected to Funk-Server")
				}
			}

		}
	}
}

func (w *Holder) getStatsInfo(v *tracker.Tracker) *Message {
	if v.Container.Labels["funk.log.stats"] == "false" {
		logger.Get().Debugw("No stats Logging for" + v.Container.Image)
		return nil
	}
	stats := v.GetStats()

	b, err := json.Marshal(stats)
	if err != nil {
		logger.Get().Errorw("Error by Marshal stats:" + err.Error())
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
		logger.Get().Debugw("No logs Logging for " + v.Container.Image)
		return nil
	}
	logs := v.GetLogs()
	var strLogs []string

	for _, value := range logs {
		strLogs = append(strLogs, string(value))

	}

	if len(strLogs) > 0 {
		logger.Get().Debugw("Logs from " + v.Container.Image)
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
		logger.Get().Debugw("No Logs from " + v.Container.Image)
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

func (h *Holder) openSocketConn(force bool) error {
	if h.streamCon == nil || force {
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
