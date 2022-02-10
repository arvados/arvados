// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { dialogActions } from 'store/dialog/dialog-actions';
import { RootState } from 'store/store';
import { ResourceKind, extractUuidKind } from 'models/resource';
import { getResource } from 'store/resources/resources';
import { GroupContentsResourcePrefix } from 'services/groups-service/groups-service';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { ContainerRequestResource } from 'models/container-request';
import { CollectionResource } from 'models/collection';
import { ProjectResource } from 'models/project';
import { ServiceRepository } from 'services/services';
import { FilterBuilder } from 'services/api/filter-builder';
import { ListResults } from 'services/common-service/common-service';
import { RepositoryResource } from 'models/repositories';
import { SshKeyResource } from 'models/ssh-key';
import { VirtualMachinesResource } from 'models/virtual-machines';
import { UserResource, getUserDisplayName } from 'models/user';
import { LinkResource } from 'models/link';
import { KeepServiceResource } from 'models/keep-services';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import React from 'react';

export const ADVANCED_TAB_DIALOG = 'advancedTabDialog';

interface AdvancedTabDialogData {
    apiResponse: any;
    metadata: ListResults<LinkResource> | string;
    user: UserResource | string;
    pythonHeader: string;
    pythonExample: string;
    cliGetHeader: string;
    cliGetExample: string;
    cliUpdateHeader: string;
    cliUpdateExample: string;
    curlHeader: string;
    curlExample: string;
}

enum CollectionData {
    COLLECTION = 'collection',
    STORAGE_CLASSES_CONFIRMED = 'storage_classes_confirmed'
}

enum ProcessData {
    CONTAINER_REQUEST = 'container_request',
    OUTPUT_NAME = 'output_name'
}

enum ProjectData {
    GROUP = 'group',
    DELETE_AT = 'delete_at'
}

enum RepositoryData {
    REPOSITORY = 'repository',
    CREATED_AT = 'created_at'
}

enum SshKeyData {
    SSH_KEY = 'authorized_key',
    CREATED_AT = 'created_at'
}

enum VirtualMachineData {
    VIRTUAL_MACHINE = 'virtual_machine',
    CREATED_AT = 'created_at'
}

enum ResourcePrefix {
    REPOSITORIES = 'repositories',
    AUTORIZED_KEYS = 'authorized_keys',
    VIRTUAL_MACHINES = 'virtual_machines',
    KEEP_SERVICES = 'keep_services',
    USERS = 'users',
    API_CLIENT_AUTHORIZATIONS = 'api_client_authorizations',
    LINKS = 'links'
}

enum KeepServiceData {
    KEEP_SERVICE = 'keep_services',
    CREATED_AT = 'created_at'
}

enum UserData {
    USER = 'user',
    USERNAME = 'username'
}

enum ApiClientAuthorizationsData {
    API_CLIENT_AUTHORIZATION = 'api_client_authorization',
    DEFAULT_OWNER_UUID = 'default_owner_uuid'
}

enum LinkData {
    LINK = 'link',
    PROPERTIES = 'properties'
}

type AdvanceResourceKind = CollectionData | ProcessData | ProjectData | RepositoryData | SshKeyData | VirtualMachineData | KeepServiceData | ApiClientAuthorizationsData | UserData | LinkData;
type AdvanceResourcePrefix = GroupContentsResourcePrefix | ResourcePrefix;
type AdvanceResponseData = ContainerRequestResource | ProjectResource | CollectionResource | RepositoryResource | SshKeyResource | VirtualMachinesResource | KeepServiceResource | ApiClientAuthorization | UserResource | LinkResource | undefined;

export const openAdvancedTabDialog = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const kind = extractUuidKind(uuid);
        switch (kind) {
            case ResourceKind.COLLECTION:
                const { data: dataCollection, metadata: metaCollection, user: userCollection } = await dispatch<any>(getDataForAdvancedTab(uuid));
                const advanceDataCollection = advancedTabData({
                    uuid,
                    metadata: metaCollection,
                    user: userCollection,
                    apiResponseKind: collectionApiResponse,
                    data: dataCollection,
                    resourceKind: CollectionData.COLLECTION,
                    resourcePrefix: GroupContentsResourcePrefix.COLLECTION,
                    resourceKindProperty: CollectionData.STORAGE_CLASSES_CONFIRMED,
                    property: dataCollection.storageClassesConfirmed
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataCollection));
                break;
            case ResourceKind.PROCESS:
                const { data: dataProcess, metadata: metaProcess, user: userProcess } = await dispatch<any>(getDataForAdvancedTab(uuid));
                const advancedDataProcess = advancedTabData({
                    uuid,
                    metadata: metaProcess,
                    user: userProcess,
                    apiResponseKind: containerRequestApiResponse,
                    data: dataProcess,
                    resourceKind: ProcessData.CONTAINER_REQUEST,
                    resourcePrefix: GroupContentsResourcePrefix.PROCESS,
                    resourceKindProperty: ProcessData.OUTPUT_NAME,
                    property: dataProcess.outputName
                });
                dispatch<any>(initAdvancedTabDialog(advancedDataProcess));
                break;
            case ResourceKind.PROJECT:
                const { data: dataProject, metadata: metaProject, user: userProject } = await dispatch<any>(getDataForAdvancedTab(uuid));
                const advanceDataProject = advancedTabData({
                    uuid,
                    metadata: metaProject,
                    user: userProject,
                    apiResponseKind: groupRequestApiResponse,
                    data: dataProject,
                    resourceKind: ProjectData.GROUP,
                    resourcePrefix: GroupContentsResourcePrefix.PROJECT,
                    resourceKindProperty: ProjectData.DELETE_AT,
                    property: dataProject.deleteAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataProject));
                break;
            case ResourceKind.REPOSITORY:
                const dataRepository = getState().repositories.items.find(it => it.uuid === uuid);
                const advanceDataRepository = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKind: repositoryApiResponse,
                    data: dataRepository,
                    resourceKind: RepositoryData.REPOSITORY,
                    resourcePrefix: ResourcePrefix.REPOSITORIES,
                    resourceKindProperty: RepositoryData.CREATED_AT,
                    property: dataRepository!.createdAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataRepository));
                break;
            case ResourceKind.SSH_KEY:
                const dataSshKey = getState().auth.sshKeys.find(it => it.uuid === uuid);
                const advanceDataSshKey = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKind: sshKeyApiResponse,
                    data: dataSshKey,
                    resourceKind: SshKeyData.SSH_KEY,
                    resourcePrefix: ResourcePrefix.AUTORIZED_KEYS,
                    resourceKindProperty: SshKeyData.CREATED_AT,
                    property: dataSshKey!.createdAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataSshKey));
                break;
            case ResourceKind.VIRTUAL_MACHINE:
                const dataVirtualMachine = getState().virtualMachines.virtualMachines.items.find(it => it.uuid === uuid);
                const advanceDataVirtualMachine = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKind: virtualMachineApiResponse,
                    data: dataVirtualMachine,
                    resourceKind: VirtualMachineData.VIRTUAL_MACHINE,
                    resourcePrefix: ResourcePrefix.VIRTUAL_MACHINES,
                    resourceKindProperty: VirtualMachineData.CREATED_AT,
                    property: dataVirtualMachine.createdAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataVirtualMachine));
                break;
            case ResourceKind.KEEP_SERVICE:
                const dataKeepService = getState().keepServices.find(it => it.uuid === uuid);
                const advanceDataKeepService = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKind: keepServiceApiResponse,
                    data: dataKeepService,
                    resourceKind: KeepServiceData.KEEP_SERVICE,
                    resourcePrefix: ResourcePrefix.KEEP_SERVICES,
                    resourceKindProperty: KeepServiceData.CREATED_AT,
                    property: dataKeepService!.createdAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataKeepService));
                break;
            case ResourceKind.USER:
                const { resources } = getState();
                const data = getResource<UserResource>(uuid)(resources);
                const metadata = await services.linkService.list({
                    filters: new FilterBuilder()
                        .addEqual('head_uuid', uuid)
                        .getFilters()
                });
                const advanceDataUser = advancedTabData({
                    uuid,
                    metadata,
                    user: '',
                    apiResponseKind: userApiResponse,
                    data,
                    resourceKind: UserData.USER,
                    resourcePrefix: ResourcePrefix.USERS,
                    resourceKindProperty: UserData.USERNAME,
                    property: data!.username
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataUser));
                break;
            case ResourceKind.API_CLIENT_AUTHORIZATION:
                const apiClientAuthorizationResources = getState().resources;
                const dataApiClientAuthorization = getResource<ApiClientAuthorization>(uuid)(apiClientAuthorizationResources);
                const advanceDataApiClientAuthorization = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKind: apiClientAuthorizationApiResponse,
                    data: dataApiClientAuthorization,
                    resourceKind: ApiClientAuthorizationsData.API_CLIENT_AUTHORIZATION,
                    resourcePrefix: ResourcePrefix.API_CLIENT_AUTHORIZATIONS,
                    resourceKindProperty: ApiClientAuthorizationsData.DEFAULT_OWNER_UUID,
                    property: dataApiClientAuthorization!.defaultOwnerUuid
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataApiClientAuthorization));
                break;
            case ResourceKind.LINK:
                const linkResources = getState().resources;
                const dataLink = getResource<LinkResource>(uuid)(linkResources);
                const advanceDataLink = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKind: linkApiResponse,
                    data: dataLink,
                    resourceKind: LinkData.LINK,
                    resourcePrefix: ResourcePrefix.LINKS,
                    resourceKindProperty: LinkData.PROPERTIES,
                    property: dataLink!.properties
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataLink));
                break;
            default:
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Could not open advanced tab for this resource.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

const getDataForAdvancedTab = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<any>(uuid)(resources);
        const metadata = await services.linkService.list({
            filters: new FilterBuilder()
                .addEqual('head_uuid', uuid)
                .getFilters()
        });
        let user;

        if (metadata.itemsAvailable) {
            metadata.items.forEach(async (item) => {
                const {tailKind, tailUuid, properties} = item;
                properties['tail'] = tailUuid;
                try {
                    if (tailKind === ResourceKind.USER && tailUuid) {
                        user = await services.userService.get(tailUuid);
                        properties['tail'] = getUserDisplayName(user);
                    }
                } catch {};
            });
        }

        return { data, metadata, user };
    };

const initAdvancedTabDialog = (data: AdvancedTabDialogData) => dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data });

interface AdvancedTabData {
    uuid: string;
    metadata: ListResults<LinkResource> | string;
    user: UserResource | string;
    apiResponseKind: any;
    data: AdvanceResponseData;
    resourceKind: AdvanceResourceKind;
    resourcePrefix: AdvanceResourcePrefix;
    resourceKindProperty: AdvanceResourceKind;
    property: any;
}

const advancedTabData = ({ uuid, user, metadata, apiResponseKind, data, resourceKind, resourcePrefix, resourceKindProperty, property }: AdvancedTabData) => {
    return {
        uuid,
        user,
        metadata,
        apiResponse: apiResponseKind(data),
        pythonHeader: pythonHeader(resourceKind),
        pythonExample: pythonExample(uuid, resourcePrefix),
        cliGetHeader: cliGetHeader(resourceKind),
        cliGetExample: cliGetExample(uuid, resourceKind),
        cliUpdateHeader: cliUpdateHeader(resourceKind, resourceKindProperty),
        cliUpdateExample: cliUpdateExample(uuid, resourceKind, property, resourceKindProperty),
        curlHeader: curlHeader(resourceKind, resourceKindProperty),
        curlExample: curlExample(uuid, resourcePrefix, property, resourceKind, resourceKindProperty),
    };
};

const pythonHeader = (resourceKind: string) =>
    `An example python command to get a ${resourceKind} using its uuid:`;

const pythonExample = (uuid: string, resourcePrefix: string) => {
    const pythonExample = `import arvados

x = arvados.api().${resourcePrefix}().get(uuid='${uuid}').execute()`;

    return pythonExample;
};

const cliGetHeader = (resourceKind: string) =>
    `An example arv command to get a ${resourceKind} using its uuid:`;

const cliGetExample = (uuid: string, resourceKind: string) => {
    const cliGetExample = `arv ${resourceKind} get \\
  --uuid ${uuid}`;

    return cliGetExample;
};

const cliUpdateHeader = (resourceKind: string, resourceName: string) =>
    `An example arv command to update the "${resourceName}" attribute for the current ${resourceKind}:`;

const cliUpdateExample = (uuid: string, resourceKind: string, resource: string | string[], resourceName: string) => {
    const CLIUpdateCollectionExample = `arv ${resourceKind} update \\
  --uuid ${uuid} \\
  --${resourceKind} '{"${resourceName}":${JSON.stringify(resource)}}'`;

    return CLIUpdateCollectionExample;
};

const curlHeader = (resourceKind: string, resource: string) =>
    `An example curl command to update the "${resource}" attribute for the current ${resourceKind}:`;

const curlExample = (uuid: string, resourcePrefix: string, resource: string | string[], resourceKind: string, resourceName: string) => {
    const curlExample = `curl -X PUT \\
  -H "Authorization: OAuth2 $ARVADOS_API_TOKEN" \\
  --data-urlencode ${resourceKind}@/dev/stdin \\
  https://$ARVADOS_API_HOST/arvados/v1/${resourcePrefix}/${uuid} \\
  <<EOF
{
  "${resourceName}": ${JSON.stringify(resource, null, 4)}
}
EOF`;

    return curlExample;
};

const stringify = (item: string | null | number | boolean) =>
    JSON.stringify(item) || 'null';

const stringifyObject = (item: any) =>
    JSON.stringify(item, null, 2) || 'null';

const containerRequestApiResponse = (apiResponse: ContainerRequestResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, description, properties, state, requestingContainerUuid, containerUuid,
        containerCountMax, mounts, runtimeConstraints, containerImage, environment, cwd, command, outputPath, priority, expiresAt, filters, containerCount,
        useExisting, schedulingParameters, outputUuid, logUuid, outputName, outputTtl } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"created_at": "${createdAt}",
"modified_at": ${stringify(modifiedAt)},
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"name": ${stringify(name)},
"description": ${stringify(description)},
"properties": ${stringifyObject(properties)},
"state": ${stringify(state)},
"requesting_container_uuid": ${stringify(requestingContainerUuid)},
"container_uuid": ${stringify(containerUuid)},
"container_count_max": ${stringify(containerCountMax)},
"mounts": ${stringifyObject(mounts)},
"runtime_constraints": ${stringifyObject(runtimeConstraints)},
"container_image": ${stringify(containerImage)},
"environment": ${stringifyObject(environment)},
"cwd": ${stringify(cwd)},
"command": ${stringifyObject(command)},
"output_path": ${stringify(outputPath)},
"priority": ${stringify(priority)},
"expires_at": ${stringify(expiresAt)},
"filters": ${stringify(filters)},
"container_count": ${stringify(containerCount)},
"use_existing": ${stringify(useExisting)},
"scheduling_parameters": ${stringifyObject(schedulingParameters)},
"output_uuid": ${stringify(outputUuid)},
"log_uuid": ${stringify(logUuid)},
"output_name": ${stringify(outputName)},
"output_ttl": ${stringify(outputTtl)}`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const collectionApiResponse = (apiResponse: CollectionResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, description, properties, portableDataHash, replicationDesired,
        replicationConfirmedAt, replicationConfirmed, deleteAt, trashAt, isTrashed, storageClassesDesired,
        storageClassesConfirmed, storageClassesConfirmedAt, currentVersionUuid, version, preserveVersion, fileCount, fileSizeTotal } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"created_at": "${createdAt}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"portable_data_hash": ${stringify(portableDataHash)},
"replication_desired": ${stringify(replicationDesired)},
"replication_confirmed_at": ${stringify(replicationConfirmedAt)},
"replication_confirmed": ${stringify(replicationConfirmed)},
"name": ${stringify(name)},
"description": ${stringify(description)},
"properties": ${stringifyObject(properties)},
"delete_at": ${stringify(deleteAt)},
"trash_at": ${stringify(trashAt)},
"is_trashed": ${stringify(isTrashed)},
"storage_classes_desired": ${JSON.stringify(storageClassesDesired, null, 2)},
"storage_classes_confirmed": ${JSON.stringify(storageClassesConfirmed, null, 2)},
"storage_classes_confirmed_at": ${stringify(storageClassesConfirmedAt)},
"current_version_uuid": ${stringify(currentVersionUuid)},
"version": ${version},
"preserve_version": ${preserveVersion},
"file_count": ${fileCount},
"file_size_total": ${fileSizeTotal}`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const groupRequestApiResponse = (apiResponse: ProjectResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, description, groupClass, trashAt, isTrashed, deleteAt, properties, writableBy } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"created_at": "${createdAt}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"name": ${stringify(name)},
"description": ${stringify(description)},
"group_class": ${stringify(groupClass)},
"trash_at": ${stringify(trashAt)},
"is_trashed": ${stringify(isTrashed)},
"delete_at": ${stringify(deleteAt)},
"properties": ${stringifyObject(properties)},
"writable_by": ${stringifyObject(writableBy)}`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const repositoryApiResponse = (apiResponse: RepositoryResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, cloneUrls } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"name": ${stringify(name)},
"created_at": "${createdAt}",
"clone_urls": ${stringifyObject(cloneUrls)}`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const sshKeyApiResponse = (apiResponse: SshKeyResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, authorizedUserUuid, expiresAt } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"authorized_user_uuid": "${authorizedUserUuid}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"name": ${stringify(name)},
"created_at": "${createdAt}",
"expires_at": "${expiresAt}"`;
    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const virtualMachineApiResponse = (apiResponse: VirtualMachinesResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, hostname } = apiResponse;
    const response = `
"hostname": ${stringify(hostname)},
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"modified_at": ${stringify(modifiedAt)},
"created_at": "${createdAt}"`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const keepServiceApiResponse = (apiResponse: KeepServiceResource) => {
    const {
        uuid, readOnly, serviceHost, servicePort, serviceSslFlag, serviceType,
        ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid
    } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"service_host": "${serviceHost}",
"service_port": "${servicePort}",
"service_ssl_flag": "${stringify(serviceSslFlag)}",
"service_type": "${serviceType}",
"created_at": "${createdAt}",
"read_only": "${stringify(readOnly)}"`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const userApiResponse = (apiResponse: UserResource) => {
    const {
        uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid,
        email, firstName, lastName, username, isActive, isAdmin, prefs, defaultOwnerUuid,
    } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"created_at": "${createdAt}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"email": "${email}",
"first_name": "${firstName}",
"last_name": "${stringify(lastName)}",
"username": "${username}",
"is_active": "${isActive},
"is_admin": "${isAdmin},
"prefs": "${stringifyObject(prefs)},
"default_owner_uuid": "${defaultOwnerUuid},
"username": "${username}"`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const apiClientAuthorizationApiResponse = (apiResponse: ApiClientAuthorization) => {
    const {
        uuid, ownerUuid, apiToken, apiClientId, userId, createdByIpAddress, lastUsedByIpAddress,
        lastUsedAt, expiresAt, defaultOwnerUuid, scopes, updatedAt, createdAt
    } = apiResponse;
    const response = `
"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"api_token": "${stringify(apiToken)}",
"api_client_id": "${stringify(apiClientId)}",
"user_id": "${stringify(userId)}",
"created_by_ip_address": "${stringify(createdByIpAddress)}",
"last_used_by_ip_address": "${stringify(lastUsedByIpAddress)}",
"last_used_at": "${stringify(lastUsedAt)}",
"expires_at": "${stringify(expiresAt)}",
"created_at": "${stringify(createdAt)}",
"updated_at": "${stringify(updatedAt)}",
"default_owner_uuid": "${stringify(defaultOwnerUuid)}",
"scopes": "${JSON.stringify(scopes, null, 2)}"`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};

const linkApiResponse = (apiResponse: LinkResource) => {
    const {
        uuid, name, headUuid, properties, headKind, tailUuid, tailKind, linkClass,
        ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid
    } = apiResponse;
    const response = `
"uuid": "${uuid}",
"name": "${name}",
"head_uuid": "${headUuid}",
"head_kind": "${headKind}",
"tail_uuid": "${tailUuid}",
"tail_kind": "${tailKind}",
"link_class": "${linkClass}",
"owner_uuid": "${ownerUuid}",
"created_at": "${stringify(createdAt)}",
"modified_at": ${stringify(modifiedAt)},
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"properties": "${JSON.stringify(properties, null, 2)}"`;

    return <span style={{ marginLeft: '-15px' }}>{'{'} {response} {'\n'} <span style={{ marginLeft: '-15px' }}>{'}'}</span></span>;
};
