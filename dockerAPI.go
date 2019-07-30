package main

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fasibio/funk_agent/logger"
)

func getTrackingContainer(cli *client.Client, ctx context.Context) ([]types.Container, error) {
	c, err := cli.ContainerList(ctx, types.ContainerListOptions{All: false})

	if err != nil {
		return nil, err
	}

	var res []types.Container
	for _, one := range c {
		if one.Labels["funk.log"] == "false" {
			continue
		} else {
			res = append(res, one)
		}
	}
	return res, nil
}

func StartListeningForContainer(ctx context.Context, trackingContainer chan []types.Container) (*client.Client, *types.Info, error) {

	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, nil, err
	}

	res, err := getTrackingContainer(cli, ctx)
	if err != nil {
		logger.Get().Errorw("Error by getTrackingContainer: " + err.Error())
	} else {
		trackingContainer <- res
	}
	info, err := cli.Info(ctx)

	if err != nil {
		return nil, nil, err
	}
	msg, errs := cli.Events(ctx, types.EventsOptions{})
	go func() {
		for e := range errs {
			logger.Get().Errorw("Error by Events: " + e.Error())
		}
	}()

	go func() {
		for m := range msg {
			if m.Type == "container" {
				res, err := getTrackingContainer(cli, ctx)
				if err != nil {
					logger.Get().Errorw("Error by getTrackingContainer: " + err.Error())
					continue
				}
				trackingContainer <- res
			}
		}
	}()
	return cli, &info, nil
}
