// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
	"encoding/json"
)

type APIEndpoint struct {
	Method string
	Path   string
	// "new attributes" key for create/update requests
	AttrsKey string
}

var (
	EndpointConfigGet                     = APIEndpoint{"GET", "arvados/v1/config", ""}
	EndpointLogin                         = APIEndpoint{"GET", "login", ""}
	EndpointCollectionCreate              = APIEndpoint{"POST", "arvados/v1/collections", "collection"}
	EndpointCollectionUpdate              = APIEndpoint{"PATCH", "arvados/v1/collections/:uuid", "collection"}
	EndpointCollectionGet                 = APIEndpoint{"GET", "arvados/v1/collections/:uuid", ""}
	EndpointCollectionList                = APIEndpoint{"GET", "arvados/v1/collections", ""}
	EndpointCollectionProvenance          = APIEndpoint{"GET", "arvados/v1/collections/:uuid/provenance", ""}
	EndpointCollectionUsedBy              = APIEndpoint{"GET", "arvados/v1/collections/:uuid/used_by", ""}
	EndpointCollectionDelete              = APIEndpoint{"DELETE", "arvados/v1/collections/:uuid", ""}
	EndpointCollectionTrash               = APIEndpoint{"POST", "arvados/v1/collections/:uuid/trash", ""}
	EndpointCollectionUntrash             = APIEndpoint{"POST", "arvados/v1/collections/:uuid/untrash", ""}
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
	UUID         string   `json:"uuid"`
	Select       []string `json:"select"`
	IncludeTrash bool     `json:"include_trash"`
}

type UntrashOptions struct {
	UUID             string `json:"uuid"`
	EnsureUniqueName bool   `json:"ensure_unique_name"`
}

type ListOptions struct {
	ClusterID          string                 `json:"cluster_id"`
	Select             []string               `json:"select"`
	Filters            []Filter               `json:"filters"`
	Where              map[string]interface{} `json:"where"`
	Limit              int                    `json:"limit"`
	Offset             int                    `json:"offset"`
	Order              []string               `json:"order"`
	Distinct           bool                   `json:"distinct"`
	Count              string                 `json:"count"`
	IncludeTrash       bool                   `json:"include_trash"`
	IncludeOldVersions bool                   `json:"include_old_versions"`
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

type LoginOptions struct {
	ReturnTo string `json:"return_to"`        // On success, redirect to this target with api_token=xxx query param
	Remote   string `json:"remote,omitempty"` // Salt token for remote Cluster ID
	Code     string `json:"code,omitempty"`   // OAuth2 callback code
	State    string `json:"state,omitempty"`  // OAuth2 callback state
}

type API interface {
	ConfigGet(ctx context.Context) (json.RawMessage, error)
	Login(ctx context.Context, options LoginOptions) (LoginResponse, error)
	CollectionCreate(ctx context.Context, options CreateOptions) (Collection, error)
	CollectionUpdate(ctx context.Context, options UpdateOptions) (Collection, error)
	CollectionGet(ctx context.Context, options GetOptions) (Collection, error)
	CollectionList(ctx context.Context, options ListOptions) (CollectionList, error)
	CollectionProvenance(ctx context.Context, options GetOptions) (map[string]interface{}, error)
	CollectionUsedBy(ctx context.Context, options GetOptions) (map[string]interface{}, error)
	CollectionDelete(ctx context.Context, options DeleteOptions) (Collection, error)
	CollectionTrash(ctx context.Context, options DeleteOptions) (Collection, error)
	CollectionUntrash(ctx context.Context, options UntrashOptions) (Collection, error)
	ContainerCreate(ctx context.Context, options CreateOptions) (Container, error)
	ContainerUpdate(ctx context.Context, options UpdateOptions) (Container, error)
	ContainerGet(ctx context.Context, options GetOptions) (Container, error)
	ContainerList(ctx context.Context, options ListOptions) (ContainerList, error)
	ContainerDelete(ctx context.Context, options DeleteOptions) (Container, error)
	ContainerLock(ctx context.Context, options GetOptions) (Container, error)
	ContainerUnlock(ctx context.Context, options GetOptions) (Container, error)
	SpecimenCreate(ctx context.Context, options CreateOptions) (Specimen, error)
	SpecimenUpdate(ctx context.Context, options UpdateOptions) (Specimen, error)
	SpecimenGet(ctx context.Context, options GetOptions) (Specimen, error)
	SpecimenList(ctx context.Context, options ListOptions) (SpecimenList, error)
	SpecimenDelete(ctx context.Context, options DeleteOptions) (Specimen, error)
	APIClientAuthorizationCurrent(ctx context.Context, options GetOptions) (APIClientAuthorization, error)
}
