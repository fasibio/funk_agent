package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

func StartListeningForContainer(ctx context.Context, trackingContainer chan []types.Container) (*client.Client, error) {

	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	res, err := getTrackingContainer(cli, ctx)
	if err != nil {
		log.Println(err)
	} else {
		trackingContainer <- res
	}
	msg, errs := cli.Events(ctx, types.EventsOptions{})
	go func() {
		for e := range errs {
			log.Println(e)
		}
	}()

	go func() {
		for m := range msg {
			if m.Type == "container" {
				res, err := getTrackingContainer(cli, ctx)
				if err != nil {
					log.Println(err)
					continue
				}
				trackingContainer <- res
			}
		}
	}()
	return cli, nil
}
