package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fasibio/funk_agent/logger"
	"github.com/fasibio/funk_agent/tracker"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli"
	"go.uber.org/zap"
)

// Holder hold information of all needed Information after startup
type Holder struct {
	streamCon          *websocket.Conn
	Props              Props
	itSelfNamedHost    string
	client             *client.Client
	trackingContainers map[string]tracker.TrackElement
	writeToServer      Serverwriter
	GeoReader          GeoReader
}

// StatsLog is a param the type can check if it is set to the right value
type StatsLog string

// IsValidate Check current value is a valid value
func (s StatsLog) IsValidate() bool {
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
	// StatsLogAll log all statsinfo without cumulate
	StatsLogAll StatsLog = "all"
	// StatsLogCumulated log only cumulate information
	StatsLogCumulated StatsLog = "cumulated"
	// StatsLogNo do not log stats info
	StatsLogNo StatsLog = "no"
)

// Props hold all cli given information
type Props struct {
	funkServerURL      string
	InsecureSkipVerify bool
	Connectionkey      string
	LogStats           StatsLog
	SwarmMode          bool
	EnableGeoIpReader  bool
}

const (
	// ClikeyInsecureSkipVerify see description in main methode
	ClikeyInsecureSkipVerify string = "insecureSkipVerify"
	// ClikeyFunkserver see description in main methode
	ClikeyFunkserver string = "funkserver"
	// ClikeySwarmmode see description in main methode
	ClikeySwarmmode string = "swarmmode"
	// ClikeyConnectionkey see description in main methode
	ClikeyConnectionkey string = "connectionkey"
	// ClikeyLogstats see description in main methode
	ClikeyLogstats string = "logstats"
	// ClikeyLoglevel see description in main methode
	ClikeyLoglevel string = "loglevel"
	// EnableGeoIPInject allows to get an location by ip address. This will download a database from https://www.maxmind.com
	EnableGeoIPInject string = "enableGeoIPInject"
	// StatsIntervall set the second where statsinfo will be send
	StatsIntervall string = "statsintervall"
)

func main() {
	app := cli.NewApp()
	app.Name = "Funk Agent"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   ClikeyInsecureSkipVerify,
			EnvVar: "INSECURE_SKIP_VERIFY",
			Usage:  "Allow insecure serverconnections",
		},
		cli.StringFlag{
			Name:   ClikeyFunkserver,
			EnvVar: "FUNK_SERVER",
			Value:  "ws://localhost:3000",
			Usage:  "the url of the funk_server",
		},
		cli.BoolFlag{
			Name:   ClikeySwarmmode,
			EnvVar: "SWARM_MODE",
			Usage:  "Set this field if the agent runs on a swarm cluster host to optimize the outputs of metadata",
		},
		cli.StringFlag{
			Name:   ClikeyConnectionkey,
			EnvVar: "CONNECTION_KEY",
			Value:  "changeMe04cf242924f6b5f96",
			Usage:  "The connectionkey given to the funk-server to connect",
		},
		cli.StringFlag{
			Name:   ClikeyLogstats,
			EnvVar: "LOG_STATS",
			Value:  "cumulated",
			Usage:  "Log the statsinfo three values allowed all, cumulated, no",
		},
		cli.StringFlag{
			Name:   ClikeyLoglevel,
			EnvVar: "LOG_LEVEL",
			Value:  "info",
			Usage:  "debug, info, warn, error ",
		},
		cli.BoolFlag{
			Name:   EnableGeoIPInject,
			EnvVar: "ENABLE_GEO_IP_INJECT",
			Usage:  "allows to get an geo location by ip address. This will download a database from https://www.maxmind.com",
		},
		cli.StringFlag{
			Name:   StatsIntervall,
			EnvVar: "STATSINTERVALL",
			Usage:  "set the second where statsinfo will be send to server",
			Value:  "15",
		},
	}
	if err := app.Run(os.Args); err != nil {
		logger.Get().Fatalw("Global error: " + err.Error())
	}
}

func run(c *cli.Context) error {
	logger.Initialize(c.String(ClikeyLoglevel))
	statslog := StatsLog(c.String(ClikeyLogstats))

	if !statslog.IsValidate() {
		return fmt.Errorf("logstats has no valid Parameter %v", statslog)
	}

	enableGeoIPInject := c.Bool(EnableGeoIPInject)
	var georeader GeoReader
	if enableGeoIPInject {
		georeader = InitGeoIP()
	}
	holder := Holder{
		Props: Props{
			funkServerURL:      c.String(ClikeyFunkserver),
			InsecureSkipVerify: c.Bool(ClikeyInsecureSkipVerify),
			Connectionkey:      c.String(ClikeyConnectionkey),
			LogStats:           statslog,
			SwarmMode:          c.Bool(ClikeySwarmmode),
			EnableGeoIpReader:  enableGeoIPInject,
		},
		GeoReader:          georeader,
		writeToServer:      WriteToServer,
		itSelfNamedHost:    "localhost",
		trackingContainers: make(map[string]tracker.TrackElement),
	}
	err := holder.openSocketConn(false)
	for err != nil {
		err = holder.openSocketConn(false)
		logger.Get().Errorw("No connection to Server... Wait 5s and try again later")
		time.Sleep(5 * time.Second)
	}

	logger.Get().Infow("Connected to Funk-Server", "swarmmode", holder.Props.SwarmMode)
	containerChan := make(chan []types.Container, 1)
	cli, info, err := StartListeningForContainer(context.Background(), containerChan)
	if err != nil {
		panic(err)
	}
	holder.itSelfNamedHost = info.Name

	mu := sync.Mutex{}

	holder.client = cli
	go holder.updateTrackingContainer(containerChan, &mu)
	ticker := time.NewTicker(5 * time.Second)

	statsSecond, err := strconv.ParseInt(c.String(StatsIntervall), 10, 64)
	if err != nil {
		return err
	}
	if holder.Props.LogStats != StatsLogNo {
		statsTicker := time.NewTicker(time.Duration(statsSecond) * time.Second)
		go holder.uploadStatsInfromation(&mu, statsTicker)
	}
	holder.uploadTrackingInformation(&mu, ticker)
	return nil
}

func (w *Holder) uploadStatsInfromation(mu *sync.Mutex, intervall *time.Ticker) {
	for {
		for range intervall.C {
			mu.Lock()
			for _, v := range w.trackingContainers {
				w.SaveStatsInfo(v)
			}
			mu.Unlock()
		}
	}
}

// Will stock the process forever like a tcplistener
func (w *Holder) uploadTrackingInformation(mu *sync.Mutex, intervall *time.Ticker) {

	for {
		for range intervall.C {
			mu.Lock()
			for _, v := range w.trackingContainers {
				w.SaveTrackingInfo(v)
			}
			mu.Unlock()
		}
	}
}

// Will stock the process forever start it in own go routine
func (w *Holder) updateTrackingContainer(containerChan chan []types.Container, mu *sync.Mutex) {
	for {
		for c := range containerChan {
			mu.Lock()
			for _, v := range c {
				d, exist := w.trackingContainers[v.ID]
				if exist {
					d.SetContainer(v)
				} else {
					w.trackingContainers[v.ID] = tracker.NewTracker(w.client, v)
				}
			}
			mu.Unlock()
		}
	}
}

// SaveStatsInfo collect all statsinfo and send them to server
func (w *Holder) SaveStatsInfo(data tracker.TrackElement) {
	var msg []Message
	if w.Props.LogStats != StatsLogNo {
		stats := w.getStatsInfo(data)
		if stats != nil {
			msg = append(msg, *stats)
		}
	}
	if len(msg) != 0 {
		err := w.writeToServer(w.streamCon, msg)
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

// SaveTrackingInfo collect all logs and send this to the server
func (w *Holder) SaveTrackingInfo(data tracker.TrackElement) {
	stoutlog := getLoggerWithContainerInformation(logger.Get(), data.GetContainer())

	var msg []Message
	logs := w.getLogs(data)
	if logs != nil {
		msg = append(msg, *logs)
	}
	if len(msg) != 0 {
		err := w.writeToServer(w.streamCon, msg)
		if err != nil {
			stoutlog.Warnw("Error by write Data to Server" + err.Error() + " try to reconnect")

			err := w.openSocketConn(true)
			if err != nil {
				stoutlog.Warnw("Can not connect try again later: " + err.Error())
			} else {
				stoutlog.Infow("Connected to Funk-Server")
			}
		}
	}
}

func getStaticContent(v tracker.TrackElement) string {
	stoutlog := getLoggerWithContainerInformation(logger.Get(), v.GetContainer())

	staticcontent := v.GetStaticContent()
	if staticcontent == "" {
		staticcontent = "{}"
	}
	var staticcontentobj interface{}
	err := json.Unmarshal([]byte(staticcontent), &staticcontentobj)
	if err != nil {
		stoutlog.Error("Error by marshal StaticContent:" + err.Error())
		return "{}"
	}

	staticcontentstr, err := json.Marshal(staticcontentobj)
	if err != nil {
		stoutlog.Error("Error by marshal StaticContent:" + err.Error())
		return "{}"
	}
	return string(staticcontentstr)
}

func (w *Holder) getStatsInfo(v tracker.TrackElement) *Message {
	stoutlog := getLoggerWithContainerInformation(logger.Get(), v.GetContainer())

	if v.GetContainer().Labels["funk.log.stats"] == "false" {
		stoutlog.Debugw("No stats Logging for" + v.GetContainer().Names[0])
		return nil
	}
	stats := v.GetStats()

	var b []byte
	if w.Props.LogStats == StatsLogCumulated {
		stats := tracker.CumulateStatsInfo(stats)
		b, err := json.Marshal(stats)
		if err != nil {
			stoutlog.Warnw("Error by Marshal stats:"+err.Error(), "stats", stats)
			return nil
		}

		return &Message{
			Time:          time.Now(),
			Type:          MessageTypeStats,
			Data:          []string{string(b)},
			Attributes:    getFilledMessageAttributes(w, v),
			SearchIndex:   v.SearchIndex() + "_stats_cumulated",
			StaticContent: getStaticContent(v),
		}
	}

	b, err := json.Marshal(stats)
	if err != nil {
		stoutlog.Warnw("Error by Marshal stats:"+err.Error(), "stats", stats)
		return nil
	}

	return &Message{
		Time:          time.Now(),
		Type:          MessageTypeStats,
		Data:          []string{string(b)},
		Attributes:    getFilledMessageAttributes(w, v),
		SearchIndex:   v.SearchIndex() + "_stats",
		StaticContent: getStaticContent(v),
	}
}

func getFilledValue(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func getFilledMessageAttributes(holder *Holder, v tracker.TrackElement) Attributes {

	if holder.Props.SwarmMode {
		return Attributes{
			Containername: getFilledValue(v.GetContainer().Labels["com.docker.swarm.task.name"], v.GetContainer().Names[0]),
			Servicename:   v.GetContainer().Labels["com.docker.swarm.service.name"],
			Namespace:     v.GetContainer().Labels["com.docker.stack.namespace"],
			Host:          holder.itSelfNamedHost,
			ContainerID:   v.GetContainer().ImageID,
		}
	}
	return Attributes{
		Containername: v.GetContainer().Names[0],
		Host:          holder.itSelfNamedHost,
		ContainerID:   v.GetContainer().ImageID,
	}

}

func (w *Holder) injectGeoIpInformation(value, keyword string) (string, error) {
	logger.Get().Debug("All param set to get geodata from this container")
	var valueobj interface{}
	json.Unmarshal([]byte(value), &valueobj)
	values := valueobj.(map[string]interface{})
	ip := values[keyword[1:]]
	if ip == nil {
		return value, fmt.Errorf("Field %s not found", keyword)
	}
	geodata, err := w.GeoReader.GetGeoDataByIP(ip.(string))
	if err != nil {
		return value, err
	}

	type location struct {
		Lon float64 `json:"lon,omitempty"`
		Lat float64 `json:"lat,omitempty"`
	}

	values["funkgeoip.location"] = fmt.Sprintf("%v,%v", geodata.Location.Latitude, geodata.Location.Longitude)
	values["funkgeoip.location_timezone"] = geodata.Location.TimeZone
	values["funkgeoip.city_name"] = geodata.City.Names["en"]
	values["funkgeoip.postal_code"] = geodata.Postal.Code
	values["funkgeoip.accuracy_radius"] = geodata.Location.AccuracyRadius
	geoinjectdata, err := json.Marshal(values)
	return string(geoinjectdata), err
}
func getLoggerWithContainerInformation(logs *zap.SugaredLogger, container types.Container) *zap.SugaredLogger {
	return logs.With(
		"containername", container.Names[0],
	)
}
func (w *Holder) getLogs(v tracker.TrackElement) *Message {
	stoutlog := getLoggerWithContainerInformation(logger.Get(), v.GetContainer())
	if v.GetContainer().Labels["funk.log.logs"] == "false" {
		stoutlog.Debugw("No logs Logging for " + v.GetContainer().Names[0])
		return nil
	}
	logs := v.GetLogs()
	var strLogs []string

	for _, value := range logs {
		if w.Props.EnableGeoIpReader && v.GetContainer().Labels["funk.log.geodatafromip"] != "" {
			keyword := v.GetContainer().Labels["funk.log.geodatafromip"]

			injectValue, err := w.injectGeoIpInformation(string(value), keyword)
			if err != nil {
				stoutlog.Warnw("Error by inject geoip data " + err.Error())
				strLogs = append(strLogs, string(value))
			} else {
				strLogs = append(strLogs, injectValue)
			}
		} else {
			strLogs = append(strLogs, string(value))
		}
	}

	if len(strLogs) > 0 {
		logger.Get().Debugw("Logs from " + v.GetContainer().Names[0])
		return &Message{
			Time:          time.Now(),
			Type:          MessageTypeLog,
			Data:          strLogs,
			SearchIndex:   v.SearchIndex() + "_logs",
			Attributes:    getFilledMessageAttributes(w, v),
			StaticContent: getStaticContent(v),
		}
	}
	logger.Get().Debugw("No Logs from " + v.GetContainer().Names[0])
	return nil

}

func openSocketConnection(url string, connectionString string) (*websocket.Conn, error) {
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

func (w *Holder) openSocketConn(force bool) error {
	if w.streamCon == nil || force {
		d, err := openSocketConnection(w.Props.funkServerURL+"/data/subscribe", w.Props.Connectionkey)
		if err != nil {
			return err
		}
		w.streamCon = d
		// go h.handleInterrupt(&done)
		// go h.checkConnAndPoll(&conn, &done)
	}
	return nil
}
