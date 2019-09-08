package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
)

type DockerClientMock struct {
	containerList          []types.Container
	containerListSendError bool
}

func (d *DockerClientMock) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	if d.containerListSendError {
		return nil, errors.New("A mock error")
	}
	return d.containerList, nil
}

func (d *DockerClientMock) Info(ctx context.Context) (types.Info, error) {
	return types.Info{}, nil
}

func (d *DockerClientMock) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
	return make(<-chan events.Message), make(<-chan error)

}

func Test_readMessages(t *testing.T) {
	tests := []struct {
		name                  string
		sendMessage           events.Message
		wantContainer         bool
		want                  []types.Container
		setContainer          []types.Container
		dockerClientSendError bool
	}{
		{
			name: "Send message were type is not container so nothing will be returnd",
			sendMessage: events.Message{
				Type: "namespace",
			},
			setContainer:          []types.Container{},
			want:                  []types.Container{},
			wantContainer:         false,
			dockerClientSendError: false,
		},
		{
			name:                  "Send message were type is container so will be return a Container",
			dockerClientSendError: false,
			sendMessage: events.Message{
				Type: "container",
			},
			setContainer: []types.Container{
				types.Container{
					ID: "Mock",
				},
			},
			want: []types.Container{
				types.Container{
					ID: "Mock",
				},
			},
			wantContainer: true,
		},
		{
			name:                  "Send message were type is container but label is no log so will be return one of the two container",
			dockerClientSendError: false,
			sendMessage: events.Message{
				Type: "container",
			},
			setContainer: []types.Container{
				types.Container{
					ID: "Not returned",
					Labels: map[string]string{
						"funk.log": "false",
					},
				},
				types.Container{
					ID: "returned",
					Labels: map[string]string{
						"funk.log": "true",
					},
				},
			},
			want: []types.Container{
				types.Container{
					ID: "returned",
					Labels: map[string]string{
						"funk.log": "true",
					},
				},
			},
			wantContainer: true,
		},
		{
			name:                  "Send message were type is container but dockerclient returns error so will got nil as trackingContainer",
			dockerClientSendError: true,
			sendMessage: events.Message{
				Type: "container",
			},
			setContainer:  nil,
			want:          nil,
			wantContainer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := DockerClientMock{
				containerList: tt.setContainer,
			}

			msg := make(chan events.Message, 1)
			msg <- tt.sendMessage
			tracking := make(chan []types.Container)
			go readMessages(context.Background(), &cli, msg, tracking)
			if tt.wantContainer == false {
				if len(tracking) != 0 {
					t.Errorf("Want no feedback but got one %v", tracking)
				}
			} else {
				res := <-tracking
				if !reflect.DeepEqual(res, tt.want) {
					t.Errorf("have a differnt result want %v got %v", tt.want, res)
				}
			}
		})
	}
}
