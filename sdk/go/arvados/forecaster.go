// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// CheckpointNode is the representation with dependencies for the checkpoints as a graph
// this is usefull to split the dependency tree code from the specific values.
type CheckpointNode struct {
	Name string `json:"name"`

	// Dependencies are what is needed to run
	Dependencies []*CheckpointNode `json:"dependencies"`
}

// Checkpoint is an individual ckeckpoint. All times are expressed in seconds,
// we should review this decision when integrating to arvados if they make sense
// or not.
type Checkpoint struct {
	CheckpointNode
	// TimeCummulative is the time rounded to the nearest secod to run this checkpoint.
	// is used in TimeAvg() to return the average time needed for the run

	TimeAvg   float64 `json:"time_average,omitempty"`
	TimeCount int     `json:"time_count,omitempty"`
	// keep track of the max and min time for from the cache.
	TimeMin float64 `json:"time_min,omitempty"`
	TimeMax float64 `json:"time_max,omitempty"`

	TimeMinComment string `json:"time_min_comment,omitempty"`
	TimeMaxComment string `json:"time_max_comment,omitempty"`
	//In the future ScatterCummulative and Scatter Count will be used in ScatterAvg() to
	// return is the average number of Scattered, this will help to estimate costs of a run.
	//ScatterCummulative uint
	//ScatterCount       int
}

type Checkpoints struct {
	UUID        string       `json:"uuid"`
	Checkpoints []Checkpoint `json:"checkpoints"`
	//EstimatedToltalTime float64               `json:"estimated_total_time"`
}
