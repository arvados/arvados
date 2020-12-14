// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// Datapoint is the format that gets outputted to the user
type Datapoint struct {
	//UUID      string `json:"uuid"` // REVIEW: do we want each individual datapoint with a UUID

	// This is a generic "Checkpoint" name, that is derived from the container name. For Example,
	// bwamem_2902 will become checkpoint "bwamem", making this easy to agregate values

	Checkpoint string `json:"checkpoint"`
	Start1     string `json:"start_1"`
	End1       string `json:"end_1"`
	Start2     string `json:"start_2"`
	End2       string `json:"end_2"`
	Reuse      bool   `json:"reuse"`
	Legend     string `json:"legend"`
}

// ForecastDatapointsOptions will have the paramenter to fetch from the local or remote cluster
// all datapoins
//type ForecastDatapointsOptions struct {
//	ClusterID            string `json:"cluster_id"`
//	ContainerRequestUUID string `json:"container_request_uuid"`
//}

// ForecastDatapointsResponse is the format that the user gets the data.
type ForecastDatapointsResponse struct {
	Datapoints []Datapoint `json:"datapoints"`
}
