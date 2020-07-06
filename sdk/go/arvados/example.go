// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

type Example struct {
	UUID      string `json:"uuid"`
	HairStyle string `json:"hair_style"`
}

type ExampleCountOptions struct {
	ClusterID string `json:"cluster_id"`
}

type ExampleCountResponse struct {
	Count int `json:"count"`
}
