package tracker

import (
	"reflect"
	"testing"

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
