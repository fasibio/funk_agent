package tracker

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/docker/docker/api/types"
)

func TestTracker_SearchIndex(t *testing.T) {
	tests := []struct {
		name      string
		container types.Container
		want      string
	}{
		{
			name: "Container has set label funk.searchindex so this will return",
			container: types.Container{
				Labels: map[string]string{
					"funk.searchindex": "mock",
				},
			},
			want: "mock",
		},
		{
			name: "Container has label funk.searchindex no set so this will return default",
			container: types.Container{
				Labels: map[string]string{},
			},
			want: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Tracker{
				container: tt.container,
			}
			if got := tr.SearchIndex(); got != tt.want {
				t.Errorf("Tracker.SearchIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTracker_GetContainer(t *testing.T) {

	t.Run("Tests that same obj set in Struct will be returned", func(t *testing.T) {

		con := types.Container{
			ID:    "122451",
			Names: []string{"Test mock container"},
		}

		tr := &Tracker{
			container: con,
		}
		if got := tr.GetContainer(); !reflect.DeepEqual(got, con) {
			t.Errorf("Tracker.GetContainer() = %v, want %v", got, con)
		}
	})
}

func TestTracker_SetContainer(t *testing.T) {

	t.Run("Tests that same obj set per setter will be inside struct", func(t *testing.T) {

		con := types.Container{
			ID:    "122451",
			Names: []string{"Test mock container"},
		}

		tr := &Tracker{
			container: types.Container{
				ID: "Another",
			},
		}
		tr.SetContainer(con)
		if got := tr.container; !reflect.DeepEqual(got, con) {
			t.Errorf("Tracker.SetContainer() = %v, want %v", got, con)
		}
	})
}

func TestTracker_GetStats(t *testing.T) {
	tests := []struct {
		name  string
		stats *Stats
	}{
		{
			name: "Set Stats will be Correctly returned",
			stats: &Stats{
				ID: "my Stats",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Tracker{
				stats: tt.stats,
			}
			cupaloy.SnapshotT(t, tr.GetStats())
		})
	}
}

func TestTracker_GetLogs(t *testing.T) {

	t.Run("Test that set logs will we return and the set log will be reseted", func(t *testing.T) {
		logs := []TrackerLogs{
			`{"mock": "1"}`,
			`{"mock": "2"}`,
			`{"mock": "1"}`,
			`{"mock": "1"}`,
		}
		tr := &Tracker{
			logs: logs,
		}
		if got := tr.GetLogs(); !reflect.DeepEqual(got, logs) {
			t.Errorf("Tracker.GetLogs() = %v, want %v", got, logs)
		}
		if len(tr.logs) != 0 {
			t.Errorf("Strcut Logs are not cleared want 0 items but got %v", len(tr.logs))
		}
	})
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name  string
		param string
		want  bool
	}{
		{
			name:  "Give json so answer have to be true",
			param: `{"mock":"yes it is"}`,
			want:  true,
		},
		{
			name:  "Give json so answer have to be true",
			param: "String as string",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSON(tt.param); got != tt.want {
				t.Errorf("IsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

type MockDockerClient struct {
	ResultLog            string
	ResultContainerStats string
}

func (m *MockDockerClient) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	r := ioutil.NopCloser(bytes.NewReader([]byte(m.ResultLog)))
	return r, nil
}

func (m *MockDockerClient) ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error) {
	r := ioutil.NopCloser(bytes.NewReader([]byte(m.ResultContainerStats)))

	return types.ContainerStats{
		Body:   r,
		OSType: "linux",
	}, nil
}

func TestNewTracker_Logs(t *testing.T) {

	tests := []struct {
		name            string
		resultLogs      string
		resultContainer string
		container       types.Container
		want            []TrackerLogs
	}{
		{
			name:            "Check that will returns the logjson",
			resultLogs:      `{"mock":true}`,
			resultContainer: `{"mock":true}`,
			container: types.Container{
				Labels: map[string]string{},
				Names:  []string{"mocktest0"},
			},
			want: []TrackerLogs{`{"mock":true}`},
		},
		{
			name:            "Check given a string he will return a json with field message",
			resultLogs:      `this is a simple textmessage`,
			resultContainer: `{"mock":true}`,
			container: types.Container{
				Labels: map[string]string{},
				Names:  []string{"mocktest0"},
			},
			want: []TrackerLogs{`{"message":"this is a simple textmessage"}`},
		},
		{
			name:            "container have a formatRegex and this will be parsed",
			resultLogs:      "2019-08-10 [negroni] 2019-08-12T12:52:07Z | 200 |      1.591596ms | localhost:3001 | POST /graphql",
			resultContainer: `{"mock":true}`,
			container: types.Container{
				Labels: map[string]string{
					"funk.log.formatRegex": `\[[a-z]*\] (?P<time>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z) \| (?P<status>[0-9]{3}) \| * (?P<request_ms>\d.\d+).....(?P<domain>.*)...(?P<method>(GET|POST|PUT|DELETE)) (?P<message>.*)`,
				},
				Names: []string{"mocktest0"},
			},
			want: []TrackerLogs{`{"domain":"localhost:3001","message":"/graphql","method":"POST","request_ms":"1.591596","status":"200","time":"2019-08-12T12:52:07Z"}`},
		},
		{
			name:            "container have a formatRegex and this will be parsed but text will not match so it will return the fallback",
			resultLogs:      "2019-08-10 i Am not parsing",
			resultContainer: `{"mock":true}`,
			container: types.Container{
				Labels: map[string]string{
					"funk.log.formatRegex": `\[[a-z]*\] (?P<time>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z) \| (?P<status>[0-9]{3}) \| * (?P<request_ms>\d.\d+).....(?P<domain>.*)...(?P<method>(GET|POST|PUT|DELETE)) (?P<message>.*)`,
				},
				Names: []string{"mocktest0"},
			},
			want: []TrackerLogs{`{"message":"i Am not parsing"}`},
		},
		{
			name:            "container have a formatRegex and this will be parsed but text will not match but is a json so it will return the json",
			resultLogs:      `2019-08-16 {"mock":true}`,
			resultContainer: `{"mock":true}`,
			container: types.Container{
				Labels: map[string]string{
					"funk.log.formatRegex": `\[[a-z]*\] (?P<time>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z) \| (?P<status>[0-9]{3}) \| * (?P<request_ms>\d.\d+).....(?P<domain>.*)...(?P<method>(GET|POST|PUT|DELETE)) (?P<message>.*)`,
				},
				Names: []string{"mocktest0"},
			},
			want: []TrackerLogs{`{"mock":true}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := MockDockerClient{
				ResultLog:            tt.resultLogs,
				ResultContainerStats: tt.resultContainer,
			}
			tracker := NewTracker(&mockClient, tt.container)
			time.Sleep(10 * time.Millisecond)
			logs := tracker.GetLogs()
			if !reflect.DeepEqual(logs, tt.want) {
				t.Errorf("Logs are different got %v want %v", logs, tt.want)

			}
		})
	}
}
