package main

import (
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/fasibio/funk_agent/tracker"
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
		tracker tracker.Tracker
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
			tracker: tracker.Tracker{
				Container: types.Container{
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
			tracker: tracker.Tracker{
				Container: types.Container{
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
			tracker: tracker.Tracker{
				Container: types.Container{
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
			if got := getFilledMessageAttributes(&tt.holder, &tt.tracker); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFilledMessageAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}
