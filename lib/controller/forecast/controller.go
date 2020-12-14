// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package forecast

import (
	"context"
	"fmt"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// New returns a new Controller for a cluster and parent API
func New(cluster *arvados.Cluster, parent arvados.API) *Controller {
	return &Controller{
		cluster: cluster,
		parent:  parent,
	}
}

// Controller Is the main object to implement the Forecast endpoints
type Controller struct {
	cluster *arvados.Cluster
	parent  arvados.API
}

func getDatapointsFromSlice(crs []arvados.ContainerRequest) Datapoints {
	var returnMap Datapoints
	returnMap = make(Datapoints, len(crs))

	for key, cr := range crs {

		if returnMap[cr.Name] == nil {
			x := Datapoint{
				ContainerRequest:     &crs[key],
				ContainerRequestUUID: cr.UUID,
				CheckpointName:       cr.Name,
				ContainerUUID:        cr.ContainerUUID,
			}
			returnMap[cr.Name] = &x
		}
	}
	return returnMap
}

// ChildContainerRequests will get all the leaves from the container request uuid
func (ctrl *Controller) ChildContainerRequests(ctx context.Context, uuid string) ([]arvados.ContainerRequest, error) {
	var crs []arvados.ContainerRequest
	var crsl arvados.ContainerRequestList
	limit := int64(100)
	offset := int64(0)

	//err := config.Arv.Call("GET", "container_requests", "", uuid, nil, &cr)
	cr, err := ctrl.parent.ContainerRequestGet(ctx, arvados.GetOptions{UUID: uuid})

	if err != nil {
		return nil, err
	}

	var filters []arvados.Filter
	filters = append(filters, arvados.Filter{Attr: "requesting_container_uuid", Operator: "=", Operand: cr.ContainerUUID})

	for len(crsl.Items) != 0 || offset == 0 {
		//err = config.Arv.List("container_requests", arvadosclient.Dict{"filters": filters, "limit": limit, "offset": offset}, &crsl)
		crsl, err = ctrl.parent.ContainerRequestList(ctx, arvados.ListOptions{Filters: filters, Limit: limit, Offset: offset})

		if err != nil {
			return nil, err
		}

		crs = append(crs, crsl.Items...)
		offset += limit
	}
	return crs, nil

}

// transform will be used to transform a input type of a forecast.Datapoint into the output type arvados.Datapoint that
// will be sent to the client later.
func transform(input *Datapoint) (output arvados.Datapoint) {
	var extraInfo string
	if input.Reuse() {
		extraInfo += `Reused`
	}

	d, err := input.Duration()
	if err == nil {
		extraInfo += fmt.Sprintf("Container duration: %s\n", d)
	}

	legend := fmt.Sprintf(`<p>%s</p><p>Container Request: %s</a></p><p>%s</p>`,
		input.CheckpointName,
		input.ContainerRequest.UUID,
		extraInfo)

	// ContainerRequest doesn't have an "end time", so we behave differently it it was reused and when it was not.
	var endTimeContainerRequest string
	if input.Reuse() {
		endTimeContainerRequest = input.ContainerRequest.CreatedAt.Format("2006-01-02 15:04:05.000 -0700")
	}

	var endTimeContainer string
	if input.Container.FinishedAt != nil {
		endTimeContainerRequest = input.Container.FinishedAt.Format("2006-01-02 15:04:05.000 -0700")
		endTimeContainer = input.Container.FinishedAt.Format("2006-01-02 15:04:05.000 -0700")
	} else { // we havent finished
		endTimeContainerRequest = time.Now().Format("2006-01-02 15:04:05.000 -0700")
		endTimeContainer = time.Now().Format("2006-01-02 15:04:05.000 -0700")
	}

	var startTimeContainer string
	if input.Container.StartedAt != nil {
		startTimeContainer = input.Container.StartedAt.Format("2006-01-02 15:04:05.000 -0700")
	} else { //container has not even started.
		startTimeContainer = time.Now().Format("2006-01-02 15:04:05.000 -0700")
	}

	output.Checkpoint = input.CheckpointName
	output.Start1 = input.ContainerRequest.CreatedAt.Format("2006-01-02 15:04:05.000 -0700")
	output.End1 = endTimeContainerRequest
	output.Start2 = startTimeContainer
	output.End2 = endTimeContainer
	output.Reuse = input.Reuse()
	output.Legend = legend

	return
}

// ForecastDatapoints returns the datapoints we have stored in the database
// for a Container Request UUID. This will follow the specs described in
// https://dev.arvados.org/projects/arvados/wiki/API_HistoricalForcasting_data_for_CR
func (ctrl *Controller) ForecastDatapoints(ctx context.Context, opts arvados.GetOptions) (resp arvados.ForecastDatapointsResponse, err error) {

	crs, err := ctrl.ChildContainerRequests(ctx, opts.UUID)
	if err != nil {
		return
	}

	datapoints := getDatapointsFromSlice(crs)

	// FIXME do this with channels in parallel a
	for _, datapoint := range datapoints {
		datapoint.Hydrate(ctx, *ctrl)
		resp.Datapoints = append(resp.Datapoints, transform(datapoint))
	}

	return
}
