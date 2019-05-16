// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

type APIEndpoint struct {
	Method string
	Path   string
	// "new attributes" key for create/update requests
	AttrsKey string
}

var (
	EndpointCollectionCreate              = APIEndpoint{"POST", "arvados/v1/collections", "collection"}
	EndpointCollectionUpdate              = APIEndpoint{"PATCH", "arvados/v1/collections/:uuid", "collection"}
	EndpointCollectionGet                 = APIEndpoint{"GET", "arvados/v1/collections/:uuid", ""}
	EndpointCollectionList                = APIEndpoint{"GET", "arvados/v1/collections", ""}
	EndpointCollectionProvenance          = APIEndpoint{"GET", "arvados/v1/collections/:uuid/provenance", ""}
	EndpointCollectionUsedBy              = APIEndpoint{"GET", "arvados/v1/collections/:uuid/used_by", ""}
	EndpointCollectionDelete              = APIEndpoint{"DELETE", "arvados/v1/collections/:uuid", ""}
	EndpointSpecimenCreate                = APIEndpoint{"POST", "arvados/v1/specimens", "specimen"}
	EndpointSpecimenUpdate                = APIEndpoint{"PATCH", "arvados/v1/specimens/:uuid", "specimen"}
	EndpointSpecimenGet                   = APIEndpoint{"GET", "arvados/v1/specimens/:uuid", ""}
	EndpointSpecimenList                  = APIEndpoint{"GET", "arvados/v1/specimens", ""}
	EndpointSpecimenDelete                = APIEndpoint{"DELETE", "arvados/v1/specimens/:uuid", ""}
	EndpointContainerCreate               = APIEndpoint{"POST", "arvados/v1/containers", "container"}
	EndpointContainerUpdate               = APIEndpoint{"PATCH", "arvados/v1/containers/:uuid", "container"}
	EndpointContainerGet                  = APIEndpoint{"GET", "arvados/v1/containers/:uuid", ""}
	EndpointContainerList                 = APIEndpoint{"GET", "arvados/v1/containers", ""}
	EndpointContainerDelete               = APIEndpoint{"DELETE", "arvados/v1/containers/:uuid", ""}
	EndpointContainerLock                 = APIEndpoint{"POST", "arvados/v1/containers/:uuid/lock", ""}
	EndpointContainerUnlock               = APIEndpoint{"POST", "arvados/v1/containers/:uuid/unlock", ""}
	EndpointAPIClientAuthorizationCurrent = APIEndpoint{"GET", "arvados/v1/api_client_authorizations/current", ""}
)

type GetOptions struct {
	UUID   string   `json:"uuid"`
	Select []string `json:"select"`
}

type ListOptions struct {
	Select  []string               `json:"select"`
	Filters []Filter               `json:"filters"`
	Where   map[string]interface{} `json:"where"`
	Limit   int                    `json:"limit"`
	Offset  int                    `json:"offset"`
	Order   []string               `json:"order"`
	Count   string                 `json:"count"`
}

type CreateOptions struct {
	ClusterID        string                 `json:"cluster_id"`
	EnsureUniqueName bool                   `json:"ensure_unique_name"`
	Select           []string               `json:"select"`
	Attrs            map[string]interface{} `json:"attrs"`
}

type UpdateOptions struct {
	UUID  string                 `json:"uuid"`
	Attrs map[string]interface{} `json:"attrs"`
}

type DeleteOptions struct {
	UUID string `json:"uuid"`
}
