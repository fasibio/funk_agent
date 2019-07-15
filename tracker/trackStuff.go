package tracker

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Tracker struct {
	Container types.Container
	Ctx       context.Context
	Client    *client.Client
	stats     *Stats
	logs      []TrackerLogs
}

func (t *Tracker) SearchIndex() string {
	index := t.Container.Labels["funk.searchindex"]
	if index == "" {
		return "default"
	}
	return index
}

type TrackerLogs string

func NewTracker(client *client.Client, container types.Container) *Tracker {
	res := &Tracker{
		Client:    client,
		Container: container,
		stats:     new(Stats),
		Ctx:       context.Background(),
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

func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil

}

func (t *Tracker) readLogs() {
	clogs, err := t.Client.ContainerLogs(t.Ctx, t.Container.ID, types.ContainerLogsOptions{
		Details:    false,
		Follow:     true,
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
	})
	if err != nil {
		log.Println(err)
	}
	defer clogs.Close()

	r := bufio.NewScanner(clogs)
	for r.Scan() {

		te := r.Text()
		if isJSON(te) {
			t.logs = append(t.logs, TrackerLogs(te))
		} else {
			te = strings.SplitN(te, " ", 2)[1]
			if isJSON(te) {
				t.logs = append(t.logs, TrackerLogs(te))
			} else {
				type fallback struct {
					Message string `json:"message,omitempty"`
				}

				fallbackMessage := fallback{
					Message: te,
				}
				bfallBack, err := json.Marshal(fallbackMessage)
				if err != nil {
					log.Println("Error by parsing fallback Message")
					continue
				}
				t.logs = append(t.logs, TrackerLogs(bfallBack))
			}

		}
	}
}

func (t *Tracker) streamStats() {
	cstats, err := t.Client.ContainerStats(t.Ctx, t.Container.ID, true)

	if err != nil {
		log.Println(err)
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

type Stats struct {
	Read         string       `json:"read"`
	Preread      string       `json:"preread"`
	PidsStats    PidsStats    `json:"pids_stats"`
	BlkioStats   BlkioStats   `json:"blkio_stats"`
	NumProcs     int64        `json:"num_procs"`
	StorageStats StorageStats `json:"storage_stats"`
	CPUStats     CPUStats     `json:"cpu_stats"`
	PrecpuStats  CPUStats     `json:"precpu_stats"`
	MemoryStats  MemoryStats  `json:"memory_stats"`
	Name         string       `json:"name"`
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
