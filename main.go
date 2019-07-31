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
	SwarmMode          bool
}

const (
	Clikey_InsecureSkipVerify string = "insecureSkipVerify"
	Clikey_Funkserver         string = "funkserver"
	Clikey_Swarmmode          string = "swarmmode"
	Clikey_Connectionkey      string = "connectionkey"
	Clikey_Logstats           string = "logstats"
	Clikey_Loglevel           string = "loglevel"
)

func main() {
	app := cli.NewApp()
	app.Name = "Funk Agent"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   Clikey_InsecureSkipVerify,
			EnvVar: "INSECURE_SKIP_VERIFY",
			Usage:  "Allow insecure serverconnections",
		},
		cli.StringFlag{
			Name:   Clikey_Funkserver,
			EnvVar: "FUNK_SERVER",
			Value:  "ws://localhost:3000",
			Usage:  "the url of the funk_server",
		},
		cli.BoolFlag{
			Name:   Clikey_Swarmmode,
			EnvVar: "SWARM_MODE",
			Usage:  "Set this field if the agent runs on a swarm cluster host to optimize the outputs of metadata",
		},
		cli.StringFlag{
			Name:   Clikey_Connectionkey,
			EnvVar: "CONNECTION_KEY",
			Value:  "changeMe04cf242924f6b5f96",
			Usage:  "The connectionkey given to the funk-server to connect",
		},
		cli.StringFlag{
			Name:   Clikey_Logstats,
			EnvVar: "LOG_STATS",
			Value:  "all",
			Usage:  "Log the statsinfo three values allowed all, cumulated (not supported now), no",
		},
		cli.StringFlag{
			Name:   Clikey_Loglevel,
			EnvVar: "LOG_LEVEL",
			Value:  "info",
			Usage:  "Log the statsinfo three values allowed all, cumulated (not supported now), no",
		},
	}
	if err := app.Run(os.Args); err != nil {
		logger.Get().Fatalw("Global error: " + err.Error())
	}
}

func run(c *cli.Context) error {
	logger.Initialize(c.String(Clikey_Loglevel))
	statslog := StatsLog(c.String(Clikey_Logstats))
	if !statslog.isValidate() {
		return fmt.Errorf("logstats has no valid Parameter" + c.String(Clikey_Logstats))
	}

	holder := Holder{
		Props: Props{
			FunkServerUrl:      c.String(Clikey_Funkserver),
			InsecureSkipVerify: c.Bool(Clikey_InsecureSkipVerify),
			Connectionkey:      c.String(Clikey_Connectionkey),
			LogStats:           statslog,
			SwarmMode:          c.Bool(Clikey_Swarmmode),
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

	if holder.Props.SwarmMode {
		logger.Get().Infow("Connected to Funk-Server with Swarm Mode")
	} else {
		logger.Get().Infow("Connected to Funk-Server")
	}
	containerChan := make(chan []types.Container, 1)
	cli, info, err := StartListeningForContainer(context.Background(), containerChan)
	if err != nil {
		panic(err)
	}
	holder.itSelfNamedHost = info.Name

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
			for _, v := range holder.trackingContainers {
				holder.SaveTrackingInfo(v)
			}
			mu.Unlock()
		}
	}

}

func (w *Holder) SaveTrackingInfo(data *tracker.Tracker) {
	var msg []Message
	logs := w.getLogs(data)
	if logs != nil {
		msg = append(msg, *logs)
	}
	if w.Props.LogStats == StatsLogAll {
		stats := w.getStatsInfo(data)
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

func (w *Holder) getStatsInfo(v *tracker.Tracker) *Message {
	if v.Container.Labels["funk.log.stats"] == "false" {
		logger.Get().Debugw("No stats Logging for" + v.Container.Names[0])
		return nil
	}
	stats := v.GetStats()

	b, err := json.Marshal(stats)
	if err != nil {
		logger.Get().Errorw("Error by Marshal stats:" + err.Error())
		return nil
	}

	return &Message{
		Time:        time.Now(),
		Type:        MessageType_Stats,
		Data:        []string{string(b)},
		Attributes:  getFilledMessageAttributes(w, v),
		SearchIndex: v.SearchIndex() + "_stats",
	}

}

func getFilledMessageAttributes(holder *Holder, v *tracker.Tracker) Attributes {
	if holder.Props.SwarmMode {
		return Attributes{
			Containername: v.Container.Labels["com.docker.swarm.task.name"],
			Servicename:   v.Container.Labels["com.docker.swarm.service.name"],
			Namespace:     v.Container.Labels["com.docker.stack.namespace"],
			Host:          holder.itSelfNamedHost,
			ContainerID:   v.Container.ImageID,
		}
	}
	return Attributes{
		Containername: v.Container.Names[0],
		Host:          holder.itSelfNamedHost,
		ContainerID:   v.Container.ImageID,
	}

}

func (w *Holder) getLogs(v *tracker.Tracker) *Message {
	if v.Container.Labels["funk.log.logs"] == "false" {
		logger.Get().Debugw("No logs Logging for " + v.Container.Names[0])
		return nil
	}
	logs := v.GetLogs()
	var strLogs []string

	for _, value := range logs {
		strLogs = append(strLogs, string(value))

	}

	if len(strLogs) > 0 {
		logger.Get().Debugw("Logs from " + v.Container.Names[0])
		return &Message{
			Time:        time.Now(),
			Type:        MessageType_Log,
			Data:        strLogs,
			SearchIndex: v.SearchIndex() + "_logs",
			Attributes:  getFilledMessageAttributes(w, v),
		}
	} else {
		logger.Get().Debugw("No Logs from " + v.Container.Names[0])
		return nil
	}
}

func openSocketConnection(url string, h *Holder, isConnOpen *bool, connectionString string) (*websocket.Conn, error) {
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
		conn := true
		d, err := openSocketConnection(h.Props.FunkServerUrl+"/data/subscribe", h, &conn, h.Props.Connectionkey)
		if err != nil {
			return err
		}
		h.streamCon = d
		// go h.handleInterrupt(&done)
		// go h.checkConnAndPoll(&conn, &done)
	}
	return nil
}
