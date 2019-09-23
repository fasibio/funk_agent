package tracker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/fasibio/funk_agent/logger"
	"go.uber.org/zap"
)

type DockerClient interface {
	ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
}

type TrackElement interface {
	SearchIndex() string
	GetStats() Stats
	GetLogs() []TrackerLogs
	GetContainer() types.Container
	SetContainer(con types.Container)
}

type Tracker struct {
	container types.Container
	ctx       context.Context
	client    DockerClient
	stats     *Stats
	logs      []TrackerLogs
}

func (t *Tracker) GetContainer() types.Container {
	return t.container
}

func (t *Tracker) SetContainer(con types.Container) {
	t.container = con
}

func (t *Tracker) SearchIndex() string {
	index := t.container.Labels["funk.searchindex"]
	if index == "" {
		return "default"
	}
	return index
}

type TrackerLogs string
type fallback struct {
	Message string `json:"message,omitempty"`
}

func NewTracker(client DockerClient, container types.Container) *Tracker {
	res := &Tracker{
		client:    client,
		container: container,
		stats:     new(Stats),
		ctx:       context.Background(),
	}
	res.runAsyncTasks()
	return res
}

func (t *Tracker) GetStats() Stats {
	return *t.stats
}
func (t *Tracker) GetLogs() []TrackerLogs {
	res := t.logs
	t.logs = make([]TrackerLogs, 0)
	return res
}

func (t *Tracker) runAsyncTasks() {
	go t.streamStats()
	go t.readLogs()
}

func IsJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

var startDate time.Time = time.Now()

func (t *Tracker) readLogs() {
	logs := getLoggerWithContainerInformation(logger.Get(), &t.container)

	clogs, err := t.client.ContainerLogs(t.ctx, t.container.ID, types.ContainerLogsOptions{
		Details:    false,
		Follow:     true,
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
		Since:      startDate.Format("2006-01-02T15:04:05"),
	})
	if err != nil {
		logs.Errorw("Error Read containerlogs:" + err.Error())
		return
	}
	defer clogs.Close()

	r := bufio.NewScanner(clogs)
	for r.Scan() {
		te := r.Text()
		var track TrackerLogs
		var err error
		if t.container.Labels["funk.log.formatRegex"] != "" {
			track, err = getTrackerLogsByFormat(t.container.Labels["funk.log.formatRegex"], strings.Trim(strings.SplitN(te, " ", 2)[1], " "))
			if err != nil {
				logs.Errorw(err.Error())
			}
			te = string(track)
		}
		track, err = getTrackerLog(te)
		if err != nil {

			logs.Errorw("Use fallback" + err.Error())
			fallbackMessage := fallback{
				Message: te,
			}
			bfallBack, _ := json.Marshal(fallbackMessage)
			track = TrackerLogs(bfallBack)

		}
		t.logs = append(t.logs, track)
	}
}

func getTrackerLogsByFormat(format, text string) (TrackerLogs, error) {
	pattern, err := regexp.Compile(format)
	if err != nil {
		return TrackerLogs(text), errors.New("Error by Parsing Format: " + err.Error())
	}
	if !pattern.MatchString(text) {
		return TrackerLogs(text), errors.New("Text does not match with given format. Use all as message Text:" + text + " Format:" + format)
	}
	body := make(map[string]interface{})
	for _, submatches := range pattern.FindAllStringSubmatchIndex(text, -1) {
		for _, one := range pattern.SubexpNames() {
			if one != "" {
				var result []byte
				result = pattern.ExpandString(result, "$"+one, text, submatches)
				body[one] = string(result)
			}
		}
	}
	res, err := json.Marshal(&body)
	if err != nil {
		return TrackerLogs(text), err
	}
	return TrackerLogs(res), nil

}

func getTrackerLog(text string) (TrackerLogs, error) {
	te := text
	if IsJSON(te) {
		return TrackerLogs(te), nil
	}
	teArray := strings.SplitN(te, " ", 2)
	if len(teArray) > 1 {
		te = teArray[1]
	}
	if IsJSON(te) {
		return TrackerLogs(te), nil
	}

	return TrackerLogs(te), errors.New("No JSON will use all as message")

}

func (t *Tracker) streamStats() {
	cstats, err := t.client.ContainerStats(t.ctx, t.container.ID, true)

	if err != nil {
		logger.Get().Errorw("Error get ContainerStats:" + err.Error())
	}
	d := json.NewDecoder(cstats.Body)
	defer cstats.Body.Close()
	var data Stats
	for d.More() {
		err := d.Decode(&data)
		if err == nil {
			t.stats = &data
		}
	}
	t.stats = new(Stats)
}

func getLoggerWithContainerInformation(logs *zap.SugaredLogger, container *types.Container) *zap.SugaredLogger {
	return logs.With(
		"containername", container.Names[0],
	)
}

func (v *Stats) calculateCPUPercentUnix() float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PrecpuStats.CPUUsage.TotalUsage)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemCPUUsage) - float64(v.PrecpuStats.SystemCPUUsage)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func CumulateStatsInfo(stats Stats) CumulateStats {
	return CumulateStats{
		RamUsageMb:      float64(stats.MemoryStats.Usage) / 1000000,
		RamUsagePercent: (float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit)) * 100,
		CPUUsagePercent: stats.calculateCPUPercentUnix(),
		RamLimitMb:      float64(stats.MemoryStats.Limit) / 1000000,
		NetIOReceiveMb:  float64(stats.Networks.Eth0.RxBytes) / 1000000,
		NetIOTransmitMb: float64(stats.Networks.Eth0.TxBytes) / 1000000,
	}
}

type CumulateStats struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent,omitempty"`
	RamUsagePercent float64 `json:"ram_usage_percent,omitempty"`
	RamUsageMb      float64 `json:"ram_usage_mb,omitempty"`
	RamLimitMb      float64 `json:"ram_limit_mb,omitempty"`
	NetIOReceiveMb  float64 `json:"net_io_usage_mb,omitempty"`
	NetIOTransmitMb float64 `json:"net_io_transmit_mb,omitempty"`
}

type Stats struct {
	Read      string    `json:"read"`
	Preread   string    `json:"preread"`
	PidsStats PidsStats `json:"pids_stats"`
	// BlkioStats   BlkioStats   `json:"blkio_stats"` // logs urgly not logable out of the box necessary ?
	NumProcs     int64        `json:"num_procs"`
	StorageStats StorageStats `json:"storage_stats"`
	CPUStats     CPUStats     `json:"cpu_stats"`
	PrecpuStats  CPUStats     `json:"precpu_stats"`
	MemoryStats  MemoryStats  `json:"memory_stats"`
	ID           string       `json:"id"`
	Networks     Networks     `json:"networks"`
}

type BlkioStats struct {
	IoServiceBytesRecursive []interface{} `json:"io_service_bytes_recursive"`
	IoServicedRecursive     []interface{} `json:"io_serviced_recursive"`
	IoQueueRecursive        []interface{} `json:"io_queue_recursive"`
	IoServiceTimeRecursive  []interface{} `json:"io_service_time_recursive"`
	IoWaitTimeRecursive     []interface{} `json:"io_wait_time_recursive"`
	IoMergedRecursive       []interface{} `json:"io_merged_recursive"`
	IoTimeRecursive         []interface{} `json:"io_time_recursive"`
	SectorsRecursive        []interface{} `json:"sectors_recursive"`
}

type CPUStats struct {
	CPUUsage       CPUUsage       `json:"cpu_usage"`
	SystemCPUUsage int64          `json:"system_cpu_usage"`
	OnlineCpus     int64          `json:"online_cpus"`
	ThrottlingData ThrottlingData `json:"throttling_data"`
}

type CPUUsage struct {
	TotalUsage        int64   `json:"total_usage"`
	PercpuUsage       []int64 `json:"percpu_usage"`
	UsageInKernelmode int64   `json:"usage_in_kernelmode"`
	UsageInUsermode   int64   `json:"usage_in_usermode"`
}

type ThrottlingData struct {
	Periods          int64 `json:"periods"`
	ThrottledPeriods int64 `json:"throttled_periods"`
	ThrottledTime    int64 `json:"throttled_time"`
}

type MemoryStats struct {
	Usage    int64            `json:"usage"`
	MaxUsage int64            `json:"max_usage"`
	Stats    map[string]int64 `json:"stats"`
	Limit    int64            `json:"limit"`
}

type Networks struct {
	Eth0 Eth0 `json:"eth0"`
}

type Eth0 struct {
	RxBytes   int64 `json:"rx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	RxErrors  int64 `json:"rx_errors"`
	RxDropped int64 `json:"rx_dropped"`
	TxBytes   int64 `json:"tx_bytes"`
	TxPackets int64 `json:"tx_packets"`
	TxErrors  int64 `json:"tx_errors"`
	TxDropped int64 `json:"tx_dropped"`
}

type PidsStats struct {
	Current int64 `json:"current"`
}

type StorageStats struct {
}
