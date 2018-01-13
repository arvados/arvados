// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Node is an arvados#node resource.
type Node struct {
	UUID       string         `json:"uuid"`
	Domain     string         `json:"domain"`
	Hostname   string         `json:"hostname"`
	IPAddress  string         `json:"ip_address"`
	LastPingAt *time.Time     `json:"last_ping_at,omitempty"`
	SlotNumber int            `json:"slot_number"`
	Status     string         `json:"status"`
	JobUUID    string         `json:"job_uuid,omitempty"`
	Properties NodeProperties `json:"properties"`
}

type NodeProperties struct {
	CloudNode      NodePropertiesCloudNode `json:"cloud_node"`
	TotalCPUCores  int                     `json:"total_cpu_cores,omitempty"`
	TotalScratchMB int64                   `json:"total_scratch_mb,omitempty"`
	TotalRAMMB     int64                   `json:"total_ram_mb,omitempty"`
}

type NodePropertiesCloudNode struct {
	Size  string  `json:"size,omitempty"`
	Price float64 `json:"price"`
}

func (c Node) resourceName() string {
	return "node"
}

// NodeList is an arvados#nodeList resource.
type NodeList struct {
	Items          []Node `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
	Limit          int    `json:"limit"`
}
