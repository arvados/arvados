// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

type APIEndpoint struct {
	Method string
	Path   string
	// "new attributes" key for create/update requests
	AttrsKey string
}

var (
	EndpointConfigGet                       = APIEndpoint{"GET", "arvados/v1/config", ""}
	EndpointVocabularyGet                   = APIEndpoint{"GET", "arvados/v1/vocabulary", ""}
	EndpointDiscoveryDocument               = APIEndpoint{"GET", "discovery/v1/apis/arvados/v1/rest", ""}
	EndpointLogin                           = APIEndpoint{"GET", "login", ""}
	EndpointLogout                          = APIEndpoint{"GET", "logout", ""}
	EndpointAuthorizedKeyCreate             = APIEndpoint{"POST", "arvados/v1/authorized_keys", "authorized_key"}
	EndpointAuthorizedKeyUpdate             = APIEndpoint{"PATCH", "arvados/v1/authorized_keys/{uuid}", "authorized_key"}
	EndpointAuthorizedKeyGet                = APIEndpoint{"GET", "arvados/v1/authorized_keys/{uuid}", ""}
	EndpointAuthorizedKeyList               = APIEndpoint{"GET", "arvados/v1/authorized_keys", ""}
	EndpointAuthorizedKeyDelete             = APIEndpoint{"DELETE", "arvados/v1/authorized_keys/{uuid}", ""}
	EndpointCollectionCreate                = APIEndpoint{"POST", "arvados/v1/collections", "collection"}
	EndpointCollectionUpdate                = APIEndpoint{"PATCH", "arvados/v1/collections/{uuid}", "collection"}
	EndpointCollectionGet                   = APIEndpoint{"GET", "arvados/v1/collections/{uuid}", ""}
	EndpointCollectionList                  = APIEndpoint{"GET", "arvados/v1/collections", ""}
	EndpointCollectionProvenance            = APIEndpoint{"GET", "arvados/v1/collections/{uuid}/provenance", ""}
	EndpointCollectionUsedBy                = APIEndpoint{"GET", "arvados/v1/collections/{uuid}/used_by", ""}
	EndpointCollectionDelete                = APIEndpoint{"DELETE", "arvados/v1/collections/{uuid}", ""}
	EndpointCollectionTrash                 = APIEndpoint{"POST", "arvados/v1/collections/{uuid}/trash", ""}
	EndpointCollectionUntrash               = APIEndpoint{"POST", "arvados/v1/collections/{uuid}/untrash", ""}
	EndpointContainerCreate                 = APIEndpoint{"POST", "arvados/v1/containers", "container"}
	EndpointContainerUpdate                 = APIEndpoint{"PATCH", "arvados/v1/containers/{uuid}", "container"}
	EndpointContainerPriorityUpdate         = APIEndpoint{"POST", "arvados/v1/containers/{uuid}/update_priority", "container"}
	EndpointContainerGet                    = APIEndpoint{"GET", "arvados/v1/containers/{uuid}", ""}
	EndpointContainerList                   = APIEndpoint{"GET", "arvados/v1/containers", ""}
	EndpointContainerDelete                 = APIEndpoint{"DELETE", "arvados/v1/containers/{uuid}", ""}
	EndpointContainerLock                   = APIEndpoint{"POST", "arvados/v1/containers/{uuid}/lock", ""}
	EndpointContainerUnlock                 = APIEndpoint{"POST", "arvados/v1/containers/{uuid}/unlock", ""}
	EndpointContainerSSH                    = APIEndpoint{"POST", "arvados/v1/containers/{uuid}/ssh", ""}
	EndpointContainerSSHCompat              = APIEndpoint{"POST", "arvados/v1/connect/{uuid}/ssh", ""} // for compatibility with arvados <2.7
	EndpointContainerGatewayTunnel          = APIEndpoint{"POST", "arvados/v1/containers/{uuid}/gateway_tunnel", ""}
	EndpointContainerGatewayTunnelCompat    = APIEndpoint{"POST", "arvados/v1/connect/{uuid}/gateway_tunnel", ""} // for compatibility with arvados <2.7
	EndpointContainerRequestCreate          = APIEndpoint{"POST", "arvados/v1/container_requests", "container_request"}
	EndpointContainerRequestUpdate          = APIEndpoint{"PATCH", "arvados/v1/container_requests/{uuid}", "container_request"}
	EndpointContainerRequestGet             = APIEndpoint{"GET", "arvados/v1/container_requests/{uuid}", ""}
	EndpointContainerRequestList            = APIEndpoint{"GET", "arvados/v1/container_requests", ""}
	EndpointContainerRequestDelete          = APIEndpoint{"DELETE", "arvados/v1/container_requests/{uuid}", ""}
	EndpointContainerRequestContainerStatus = APIEndpoint{"GET", "arvados/v1/container_requests/{uuid}/container_status", ""}
	EndpointContainerRequestLog             = APIEndpoint{"GET", "arvados/v1/container_requests/{uuid}/log{path:|/.*}", ""}
	EndpointGroupCreate                     = APIEndpoint{"POST", "arvados/v1/groups", "group"}
	EndpointGroupUpdate                     = APIEndpoint{"PATCH", "arvados/v1/groups/{uuid}", "group"}
	EndpointGroupGet                        = APIEndpoint{"GET", "arvados/v1/groups/{uuid}", ""}
	EndpointGroupList                       = APIEndpoint{"GET", "arvados/v1/groups", ""}
	EndpointGroupContents                   = APIEndpoint{"GET", "arvados/v1/groups/contents", ""}
	EndpointGroupContentsUUIDInPath         = APIEndpoint{"GET", "arvados/v1/groups/{uuid}/contents", ""} // Alternative HTTP route; client-side code should always use EndpointGroupContents instead
	EndpointGroupShared                     = APIEndpoint{"GET", "arvados/v1/groups/shared", ""}
	EndpointGroupDelete                     = APIEndpoint{"DELETE", "arvados/v1/groups/{uuid}", ""}
	EndpointGroupTrash                      = APIEndpoint{"POST", "arvados/v1/groups/{uuid}/trash", ""}
	EndpointGroupUntrash                    = APIEndpoint{"POST", "arvados/v1/groups/{uuid}/untrash", ""}
	EndpointLinkCreate                      = APIEndpoint{"POST", "arvados/v1/links", "link"}
	EndpointLinkUpdate                      = APIEndpoint{"PATCH", "arvados/v1/links/{uuid}", "link"}
	EndpointLinkGet                         = APIEndpoint{"GET", "arvados/v1/links/{uuid}", ""}
	EndpointLinkList                        = APIEndpoint{"GET", "arvados/v1/links", ""}
	EndpointLinkDelete                      = APIEndpoint{"DELETE", "arvados/v1/links/{uuid}", ""}
	EndpointLogCreate                       = APIEndpoint{"POST", "arvados/v1/logs", "log"}
	EndpointLogUpdate                       = APIEndpoint{"PATCH", "arvados/v1/logs/{uuid}", "log"}
	EndpointLogGet                          = APIEndpoint{"GET", "arvados/v1/logs/{uuid}", ""}
	EndpointLogList                         = APIEndpoint{"GET", "arvados/v1/logs", ""}
	EndpointLogDelete                       = APIEndpoint{"DELETE", "arvados/v1/logs/{uuid}", ""}
	EndpointSysTrashSweep                   = APIEndpoint{"POST", "sys/trash_sweep", ""}
	EndpointUserActivate                    = APIEndpoint{"POST", "arvados/v1/users/{uuid}/activate", ""}
	EndpointUserCreate                      = APIEndpoint{"POST", "arvados/v1/users", "user"}
	EndpointUserCurrent                     = APIEndpoint{"GET", "arvados/v1/users/current", ""}
	EndpointUserDelete                      = APIEndpoint{"DELETE", "arvados/v1/users/{uuid}", ""}
	EndpointUserGet                         = APIEndpoint{"GET", "arvados/v1/users/{uuid}", ""}
	EndpointUserGetCurrent                  = APIEndpoint{"GET", "arvados/v1/users/current", ""}
	EndpointUserGetSystem                   = APIEndpoint{"GET", "arvados/v1/users/system", ""}
	EndpointUserList                        = APIEndpoint{"GET", "arvados/v1/users", ""}
	EndpointUserMerge                       = APIEndpoint{"POST", "arvados/v1/users/merge", ""}
	EndpointUserSetup                       = APIEndpoint{"POST", "arvados/v1/users/setup", "user"}
	EndpointUserSystem                      = APIEndpoint{"GET", "arvados/v1/users/system", ""}
	EndpointUserUnsetup                     = APIEndpoint{"POST", "arvados/v1/users/{uuid}/unsetup", ""}
	EndpointUserUpdate                      = APIEndpoint{"PATCH", "arvados/v1/users/{uuid}", "user"}
	EndpointUserBatchUpdate                 = APIEndpoint{"PATCH", "arvados/v1/users/batch_update", ""}
	EndpointUserAuthenticate                = APIEndpoint{"POST", "arvados/v1/users/authenticate", ""}
	EndpointAPIClientAuthorizationCurrent   = APIEndpoint{"GET", "arvados/v1/api_client_authorizations/current", ""}
	EndpointAPIClientAuthorizationCreate    = APIEndpoint{"POST", "arvados/v1/api_client_authorizations", "api_client_authorization"}
	EndpointAPIClientAuthorizationUpdate    = APIEndpoint{"PUT", "arvados/v1/api_client_authorizations/{uuid}", "api_client_authorization"}
	EndpointAPIClientAuthorizationList      = APIEndpoint{"GET", "arvados/v1/api_client_authorizations", ""}
	EndpointAPIClientAuthorizationDelete    = APIEndpoint{"DELETE", "arvados/v1/api_client_authorizations/{uuid}", ""}
	EndpointAPIClientAuthorizationGet       = APIEndpoint{"GET", "arvados/v1/api_client_authorizations/{uuid}", ""}
)

type ContainerSSHOptions struct {
	UUID          string `json:"uuid"`
	DetachKeys    string `json:"detach_keys"`
	LoginUsername string `json:"login_username"`
	NoForward     bool   `json:"no_forward"`
}

type ConnectionResponse struct {
	Conn   net.Conn           `json:"-"`
	Bufrw  *bufio.ReadWriter  `json:"-"`
	Logger logrus.FieldLogger `json:"-"`
	Header http.Header        `json:"-"`
}

type ContainerGatewayTunnelOptions struct {
	UUID       string `json:"uuid"`
	AuthSecret string `json:"auth_secret"`
}

type GetOptions struct {
	UUID         string   `json:"uuid,omitempty"`
	Select       []string `json:"select"`
	IncludeTrash bool     `json:"include_trash"`
	ForwardedFor string   `json:"forwarded_for,omitempty"`
	Remote       string   `json:"remote,omitempty"`
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
	Limit              int64                  `json:"limit"`
	Offset             int64                  `json:"offset"`
	Order              []string               `json:"order"`
	Distinct           bool                   `json:"distinct"`
	Count              string                 `json:"count"`
	IncludeTrash       bool                   `json:"include_trash"`
	IncludeOldVersions bool                   `json:"include_old_versions"`
	BypassFederation   bool                   `json:"bypass_federation"`
	ForwardedFor       string                 `json:"forwarded_for,omitempty"`
	Include            []string               `json:"include"`
}

type CreateOptions struct {
	ClusterID        string                 `json:"cluster_id"`
	EnsureUniqueName bool                   `json:"ensure_unique_name"`
	Select           []string               `json:"select"`
	Attrs            map[string]interface{} `json:"attrs"`
	// ReplaceFiles only applies when creating a collection.
	ReplaceFiles map[string]string `json:"replace_files"`
}

type UpdateOptions struct {
	UUID             string                 `json:"uuid"`
	Attrs            map[string]interface{} `json:"attrs"`
	Select           []string               `json:"select"`
	BypassFederation bool                   `json:"bypass_federation"`
	// ReplaceFiles only applies when updating a collection.
	ReplaceFiles map[string]string `json:"replace_files"`
}

type GroupContentsOptions struct {
	ClusterID          string   `json:"cluster_id"`
	UUID               string   `json:"uuid,omitempty"`
	Select             []string `json:"select"`
	Filters            []Filter `json:"filters"`
	Limit              int64    `json:"limit"`
	Offset             int64    `json:"offset"`
	Order              []string `json:"order"`
	Distinct           bool     `json:"distinct"`
	Count              string   `json:"count"`
	Include            []string `json:"include"`
	Recursive          bool     `json:"recursive"`
	IncludeTrash       bool     `json:"include_trash"`
	IncludeOldVersions bool     `json:"include_old_versions"`
	ExcludeHomeProject bool     `json:"exclude_home_project"`
}

type UserActivateOptions struct {
	UUID string `json:"uuid"`
}

type UserSetupOptions struct {
	UUID                  string                 `json:"uuid,omitempty"`
	Email                 string                 `json:"email,omitempty"`
	OpenIDPrefix          string                 `json:"openid_prefix,omitempty"`
	RepoName              string                 `json:"repo_name,omitempty"`
	VMUUID                string                 `json:"vm_uuid,omitempty"`
	SendNotificationEmail bool                   `json:"send_notification_email,omitempty"`
	Attrs                 map[string]interface{} `json:"attrs"`
}

type UserMergeOptions struct {
	NewUserUUID       string `json:"new_user_uuid,omitempty"`
	OldUserUUID       string `json:"old_user_uuid,omitempty"`
	NewOwnerUUID      string `json:"new_owner_uuid,omitempty"`
	NewUserToken      string `json:"new_user_token,omitempty"`
	RedirectToNewUser bool   `json:"redirect_to_new_user"`
}

type UserBatchUpdateOptions struct {
	Updates map[string]map[string]interface{} `json:"updates"`
}

type UserBatchUpdateResponse struct{}

type DeleteOptions struct {
	UUID string `json:"uuid"`
}

type LoginOptions struct {
	ReturnTo string `json:"return_to"`        // On success, redirect to this target with api_token=xxx query param
	Remote   string `json:"remote,omitempty"` // Salt token for remote Cluster ID
	Code     string `json:"code,omitempty"`   // OAuth2 callback code
	State    string `json:"state,omitempty"`  // OAuth2 callback state
}

type UserAuthenticateOptions struct {
	Username string `json:"username,omitempty"` // PAM username
	Password string `json:"password,omitempty"` // PAM password
}

type LogoutOptions struct {
	ReturnTo string `json:"return_to"` // Redirect to this URL after logging out
}

type BlockReadOptions struct {
	Locator      string
	WriteTo      io.Writer
	LocalLocator func(string)
}

type BlockWriteOptions struct {
	Hash           string
	Data           []byte
	Reader         io.Reader // Must be set if Data is nil.
	DataSize       int       // Must be set if Data is nil.
	RequestID      string
	StorageClasses []string
	Replicas       int
	Attempts       int
}

type BlockWriteResponse struct {
	Locator        string
	Replicas       int
	StorageClasses map[string]int
}

type WebDAVOptions struct {
	Method string
	Path   string
	Header http.Header
}

type ContainerLogOptions struct {
	UUID      string `json:"uuid"`
	NoForward bool   `json:"no_forward"`
	WebDAVOptions
}

type API interface {
	ConfigGet(ctx context.Context) (json.RawMessage, error)
	VocabularyGet(ctx context.Context) (Vocabulary, error)
	Login(ctx context.Context, options LoginOptions) (LoginResponse, error)
	Logout(ctx context.Context, options LogoutOptions) (LogoutResponse, error)
	AuthorizedKeyCreate(ctx context.Context, options CreateOptions) (AuthorizedKey, error)
	AuthorizedKeyUpdate(ctx context.Context, options UpdateOptions) (AuthorizedKey, error)
	AuthorizedKeyGet(ctx context.Context, options GetOptions) (AuthorizedKey, error)
	AuthorizedKeyList(ctx context.Context, options ListOptions) (AuthorizedKeyList, error)
	AuthorizedKeyDelete(ctx context.Context, options DeleteOptions) (AuthorizedKey, error)
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
	ContainerPriorityUpdate(ctx context.Context, options UpdateOptions) (Container, error)
	ContainerGet(ctx context.Context, options GetOptions) (Container, error)
	ContainerList(ctx context.Context, options ListOptions) (ContainerList, error)
	ContainerDelete(ctx context.Context, options DeleteOptions) (Container, error)
	ContainerLock(ctx context.Context, options GetOptions) (Container, error)
	ContainerUnlock(ctx context.Context, options GetOptions) (Container, error)
	ContainerSSH(ctx context.Context, options ContainerSSHOptions) (ConnectionResponse, error)
	ContainerGatewayTunnel(ctx context.Context, options ContainerGatewayTunnelOptions) (ConnectionResponse, error)
	ContainerRequestCreate(ctx context.Context, options CreateOptions) (ContainerRequest, error)
	ContainerRequestUpdate(ctx context.Context, options UpdateOptions) (ContainerRequest, error)
	ContainerRequestGet(ctx context.Context, options GetOptions) (ContainerRequest, error)
	ContainerRequestList(ctx context.Context, options ListOptions) (ContainerRequestList, error)
	ContainerRequestDelete(ctx context.Context, options DeleteOptions) (ContainerRequest, error)
	ContainerRequestContainerStatus(ctx context.Context, options GetOptions) (ContainerStatus, error)
	ContainerRequestLog(ctx context.Context, options ContainerLogOptions) (http.Handler, error)
	GroupCreate(ctx context.Context, options CreateOptions) (Group, error)
	GroupUpdate(ctx context.Context, options UpdateOptions) (Group, error)
	GroupGet(ctx context.Context, options GetOptions) (Group, error)
	GroupList(ctx context.Context, options ListOptions) (GroupList, error)
	GroupContents(ctx context.Context, options GroupContentsOptions) (ObjectList, error)
	GroupShared(ctx context.Context, options ListOptions) (GroupList, error)
	GroupDelete(ctx context.Context, options DeleteOptions) (Group, error)
	GroupTrash(ctx context.Context, options DeleteOptions) (Group, error)
	GroupUntrash(ctx context.Context, options UntrashOptions) (Group, error)
	LinkCreate(ctx context.Context, options CreateOptions) (Link, error)
	LinkUpdate(ctx context.Context, options UpdateOptions) (Link, error)
	LinkGet(ctx context.Context, options GetOptions) (Link, error)
	LinkList(ctx context.Context, options ListOptions) (LinkList, error)
	LinkDelete(ctx context.Context, options DeleteOptions) (Link, error)
	LogCreate(ctx context.Context, options CreateOptions) (Log, error)
	LogUpdate(ctx context.Context, options UpdateOptions) (Log, error)
	LogGet(ctx context.Context, options GetOptions) (Log, error)
	LogList(ctx context.Context, options ListOptions) (LogList, error)
	LogDelete(ctx context.Context, options DeleteOptions) (Log, error)
	SysTrashSweep(ctx context.Context, options struct{}) (struct{}, error)
	UserCreate(ctx context.Context, options CreateOptions) (User, error)
	UserUpdate(ctx context.Context, options UpdateOptions) (User, error)
	UserMerge(ctx context.Context, options UserMergeOptions) (User, error)
	UserActivate(ctx context.Context, options UserActivateOptions) (User, error)
	UserSetup(ctx context.Context, options UserSetupOptions) (map[string]interface{}, error)
	UserUnsetup(ctx context.Context, options GetOptions) (User, error)
	UserGet(ctx context.Context, options GetOptions) (User, error)
	UserGetCurrent(ctx context.Context, options GetOptions) (User, error)
	UserGetSystem(ctx context.Context, options GetOptions) (User, error)
	UserList(ctx context.Context, options ListOptions) (UserList, error)
	UserDelete(ctx context.Context, options DeleteOptions) (User, error)
	UserBatchUpdate(context.Context, UserBatchUpdateOptions) (UserList, error)
	UserAuthenticate(ctx context.Context, options UserAuthenticateOptions) (APIClientAuthorization, error)
	APIClientAuthorizationCurrent(ctx context.Context, options GetOptions) (APIClientAuthorization, error)
	APIClientAuthorizationCreate(ctx context.Context, options CreateOptions) (APIClientAuthorization, error)
	APIClientAuthorizationList(ctx context.Context, options ListOptions) (APIClientAuthorizationList, error)
	APIClientAuthorizationDelete(ctx context.Context, options DeleteOptions) (APIClientAuthorization, error)
	APIClientAuthorizationUpdate(ctx context.Context, options UpdateOptions) (APIClientAuthorization, error)
	APIClientAuthorizationGet(ctx context.Context, options GetOptions) (APIClientAuthorization, error)
	DiscoveryDocument(ctx context.Context) (DiscoveryDocument, error)
}
