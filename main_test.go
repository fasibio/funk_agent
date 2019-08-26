package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bouk/monkey"
	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/docker/docker/api/types"
	"github.com/fasibio/funk_agent/tracker"
	"github.com/gorilla/websocket"
)

func getSwarmModeLabels(containername, servicename, namespace string) map[string]string {
	res := make(map[string]string)
	res["com.docker.swarm.task.name"] = containername
	res["com.docker.swarm.service.name"] = servicename
	res["com.docker.stack.namespace"] = namespace
	return res
}
func Test_getFilledMessageAttributes(t *testing.T) {
	tests := []struct {
		name    string
		holder  Holder
		tracker tracker.TrackElement
		want    Attributes
	}{
		{
			name: "fill Attribute without Swarmmode",
			want: Attributes{
				Containername: "Containername",
				Host:          "MockTest",
				ContainerID:   "MockImageid",
			},
			holder: Holder{
				itSelfNamedHost: "MockTest",
				Props: Props{
					SwarmMode: false,
				},
			},
			tracker: &TrackerMock{
				Con: types.Container{
					ImageID: "MockImageid",
					Names: []string{
						"Containername",
					},
				},
			},
		},
		{
			name: "fill Attribute with Swarmmode all labels are set",
			want: Attributes{
				Containername: "Containername",
				Servicename:   "ServiceName",
				Namespace:     "namespace",
				Host:          "MockTest",
				ContainerID:   "MockImageid",
			},
			holder: Holder{
				itSelfNamedHost: "MockTest",
				Props: Props{
					SwarmMode: true,
				},
			},
			tracker: &TrackerMock{
				Con: types.Container{
					Labels: getSwarmModeLabels("Containername", "ServiceName", "namespace"),
					Names: []string{
						"Containername_not_used",
					},
					ImageID: "MockImageid",
				},
			},
		},
		{
			name: "fill Attribute with Swarmmode all labels without containername label are set he use the fallback container name",
			want: Attributes{
				Containername: "Containername",
				Servicename:   "ServiceName",
				Namespace:     "namespace",
				Host:          "MockTest",
				ContainerID:   "MockImageid",
			},
			holder: Holder{
				itSelfNamedHost: "MockTest",
				Props: Props{
					SwarmMode: true,
				},
			},
			tracker: &TrackerMock{
				Con: types.Container{
					Labels: getSwarmModeLabels("", "ServiceName", "namespace"),
					Names: []string{
						"Containername",
					},
					ImageID: "MockImageid",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFilledMessageAttributes(&tt.holder, tt.tracker); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFilledMessageAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

type TrackerMock struct {
	Stats tracker.Stats
	Log   tracker.TrackerLogs
	Con   types.Container
}

func (t *TrackerMock) SearchIndex() string {
	return "MockIndex"
}
func (t *TrackerMock) GetStats() tracker.Stats {
	return t.Stats
}

func (t *TrackerMock) GetLogs() []tracker.TrackerLogs {
	res := make([]tracker.TrackerLogs, 1)
	res = append(res, t.Log)
	return res
}

func (t *TrackerMock) GetContainer() types.Container {
	return t.Con
}

func (t *TrackerMock) SetContainer(con types.Container) {}

func TestHolder_SaveTrackingInfo(t *testing.T) {
	wayback := time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC)
	patch := monkey.Patch(time.Now, func() time.Time { return wayback })
	defer patch.Unpatch()
	tests := []struct {
		name                  string
		itSelfNamedHost       string
		logStats              StatsLog
		swarmMode             bool
		arg                   func() tracker.TrackElement
		writeToServerHasError bool
	}{
		{
			name:                  "Send only logs no stats container has no funk labels and no swarmmode",
			writeToServerHasError: false,
			itSelfNamedHost:       "test_unit",
			logStats:              StatsLogNo,
			swarmMode:             false,
			arg: func() tracker.TrackElement {
				res := TrackerMock{
					Log: `{"mock": "str"}`,
					Con: types.Container{
						Names:   []string{"mockContainer"},
						ImageID: "mockContainer-0001",
					},
				}
				return &res
			},
		},
		{
			name:                  "Send only logs no stats container has funk labels no logs and no swarmmode ==> so nothing will send",
			writeToServerHasError: false,
			itSelfNamedHost:       "test_unit",
			logStats:              StatsLogNo,
			swarmMode:             false,
			arg: func() tracker.TrackElement {
				res := TrackerMock{
					Log: `{"mock": "str"}`,
					Con: types.Container{
						Labels: map[string]string{
							"funk.log.logs": "false",
						},
						Names:   []string{"mockContainer"},
						ImageID: "mockContainer-0001",
					},
				}
				return &res
			},
		},
		{
			name:                  "Send only logs no stats container has no funk labels and is swarmmode",
			writeToServerHasError: false,
			itSelfNamedHost:       "test_unit",
			logStats:              StatsLogNo,
			swarmMode:             true,
			arg: func() tracker.TrackElement {
				res := TrackerMock{
					Log: `{"mock": "str"}`,
					Con: types.Container{
						Labels: map[string]string{
							"com.docker.swarm.task.name":    "com.docker.swarm.task.name",
							"com.docker.swarm.service.name": "com.docker.swarm.service.name",
							"com.docker.stack.namespace":    "com.docker.stack.namespace",
						},
						Names:   []string{"mockContainer"},
						ImageID: "mockContainer-0001",
					},
				}
				return &res
			},
		},
		{
			name:                  "Send logs and stats info container has no funk labels and is swarmmode",
			writeToServerHasError: false,
			itSelfNamedHost:       "test_unit",
			logStats:              StatsLogAll,
			swarmMode:             true,
			arg: func() tracker.TrackElement {
				res := TrackerMock{
					Log: `{"mock": "str"}`,
					Stats: tracker.Stats{
						Read:    "mock",
						Preread: "mock",
						CPUStats: tracker.CPUStats{
							SystemCPUUsage: 10,
						},
					},
					Con: types.Container{
						Labels: map[string]string{
							"com.docker.swarm.task.name":    "com.docker.swarm.task.name",
							"com.docker.swarm.service.name": "com.docker.swarm.service.name",
							"com.docker.stack.namespace":    "com.docker.stack.namespace",
						},
						Names:   []string{"mockContainer"},
						ImageID: "mockContainer-0001",
					},
				}
				return &res
			},
		},
		{
			name:                  "Send logs and stats info container has no funk labels and is swarmmode but container has label not log stats",
			writeToServerHasError: false,
			itSelfNamedHost:       "test_unit",
			logStats:              StatsLogAll,
			swarmMode:             true,
			arg: func() tracker.TrackElement {
				res := TrackerMock{
					Log: `{"mock": "str"}`,
					Stats: tracker.Stats{
						Read:    "mock",
						Preread: "mock",
						CPUStats: tracker.CPUStats{
							SystemCPUUsage: 10,
						},
					},
					Con: types.Container{
						Labels: map[string]string{
							"com.docker.swarm.task.name":    "com.docker.swarm.task.name",
							"com.docker.swarm.service.name": "com.docker.swarm.service.name",
							"com.docker.stack.namespace":    "com.docker.stack.namespace",
							"funk.log.stats":                "false",
						},
						Names:   []string{"mockContainer"},
						ImageID: "mockContainer-0001",
					},
				}
				return &res
			},
		},
		{
			name:                  "Want to Send only logs no stats container has no funk labels and no swarmmode but writeToServerSend error so try to reconnect",
			writeToServerHasError: true,
			itSelfNamedHost:       "test_unit",
			logStats:              StatsLogNo,
			swarmMode:             false,
			arg: func() tracker.TrackElement {
				res := TrackerMock{
					Log: `{"mock": "str"}`,
					Con: types.Container{
						Names:   []string{"mockContainer"},
						ImageID: "mockContainer-0001",
					},
				}
				return &res
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Holder{
				Props: Props{
					LogStats:  tt.logStats,
					SwarmMode: tt.swarmMode,
				},
				itSelfNamedHost: tt.itSelfNamedHost,
				writeToServer: func(con *websocket.Conn, msg []Message) error {
					if tt.writeToServerHasError {
						return errors.New("Mock error")
					}
					cupaloy.SnapshotT(t, msg)
					return nil
				},
			}
			w.SaveTrackingInfo(tt.arg())
		})
	}
}
