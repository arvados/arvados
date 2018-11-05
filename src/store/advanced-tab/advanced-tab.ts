// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from '~/store/dialog/dialog-actions';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { ResourceKind, extractUuidKind } from '~/models/resource';
import { getResource } from '~/store/resources/resources';
import { GroupContentsResourcePrefix } from '~/services/groups-service/groups-service';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { ContainerRequestResource } from '~/models/container-request';
import { CollectionResource } from '~/models/collection';
import { ProjectResource } from '~/models/project';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from '~/services/api/filter-builder';

export const ADVANCED_TAB_DIALOG = 'advancedTabDialog';

export interface AdvancedTabDialogData {
    apiResponse: any;
    metadata: any;
    uuid: string;
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

export const openAdvancedTabDialog = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const kind = extractUuidKind(uuid);
        const data = getResource<any>(uuid)(resources);
        const user = await services.userService.list({
            filters: new FilterBuilder()
                .addEqual('uuid', data.ownerUuid)
                .getFilters()
        });
        const metadata = await services.linkService.list({
            filters: new FilterBuilder()
                .addEqual('headUuid', uuid)
                .getFilters()
        });
        if (data) {
            if (kind === ResourceKind.COLLECTION) {
                const dataCollection: AdvancedTabDialogData = advancedTabData(uuid, metadata, user, collectionApiResponse, data, CollectionData.COLLECTION, GroupContentsResourcePrefix.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED, data.storageClassesConfirmed);
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataCollection }));
            } else if (kind === ResourceKind.PROCESS) {
                const dataProcess: AdvancedTabDialogData = advancedTabData(uuid, metadata, user, containerRequestApiResponse, data, ProcessData.CONTAINER_REQUEST, GroupContentsResourcePrefix.PROCESS, ProcessData.OUTPUT_NAME, data.outputName);
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataProcess }));
            } else if (kind === ResourceKind.PROJECT) {
                const dataProject: AdvancedTabDialogData = advancedTabData(uuid, metadata, user, groupRequestApiResponse, data, ProjectData.GROUP, GroupContentsResourcePrefix.PROJECT, ProjectData.DELETE_AT, data.deleteAt);
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataProject }));
            }
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Could not open advanced tab for this resource.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

const advancedTabData = (uuid: string, metadata: any, user: any, apiResponseKind: any, data: any, resourceKind: CollectionData | ProcessData | ProjectData, resourcePrefix: GroupContentsResourcePrefix, resourceKindProperty: CollectionData | ProcessData | ProjectData, property: any) => {
    return {
        uuid,
        metadata: { ...metadata, user },
        apiResponse: apiResponseKind(data),
        pythonHeader: pythonHeader(resourceKind),
        pythonExample: pythonExample(uuid, resourcePrefix),
        cliGetHeader: cliGetHeader(resourceKind),
        cliGetExample: cliGetExample(uuid, resourcePrefix),
        cliUpdateHeader: cliUpdateHeader(resourceKind, resourceKindProperty),
        cliUpdateExample: cliUpdateExample(uuid, resourceKind, property, resourceKind),
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

const cliGetExample = (uuid: string, resourcePrefix: string) => {
    const cliGetExample = `arv ${resourcePrefix} get \\
 --uuid ${uuid}`;

    return cliGetExample;
};

const cliUpdateHeader = (resourceKind: string, resourceName: string) =>
    `An example arv command to update the "${resourceName}" attribute for the current ${resourceKind}:`;

const cliUpdateExample = (uuid: string, resourceKind: string, resource: string | string[], resourceName: string) => {
    const CLIUpdateCollectionExample = `arv ${resourceKind} update \\ 
 --uuid ${uuid} \\
 --${resourceKind} '{"${resourceName}":${resource}}'`;

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
  "${resourceName}": ${resource}
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
    const response = `"uuid": "${uuid}",
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
"container_image": "${stringify(containerImage)}",
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

    return response;
};

const collectionApiResponse = (apiResponse: CollectionResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, description, properties, portableDataHash, replicationDesired,
        replicationConfirmedAt, replicationConfirmed, manifestText, deleteAt, trashAt, isTrashed, storageClassesDesired,
        storageClassesConfirmed, storageClassesConfirmedAt } = apiResponse;
    const response = `"uuid": "${uuid}",
"owner_uuid": "${ownerUuid}",
"created_at": "${createdAt}",
"modified_by_client_uuid": ${stringify(modifiedByClientUuid)},
"modified_by_user_uuid": ${stringify(modifiedByUserUuid)},
"modified_at": ${stringify(modifiedAt)},
"portable_data_hash": ${stringify(portableDataHash)},
"replication_desired": ${stringify(replicationDesired)},
"replication_confirmed_at": ${stringify(replicationConfirmedAt)},
"replication_confirmed": ${stringify(replicationConfirmed)},
"manifest_text": ${stringify(manifestText)},
"name": ${stringify(name)},
"description": ${stringify(description)},
"properties": ${stringifyObject(properties)},
"delete_at": ${stringify(deleteAt)},
"trash_at": ${stringify(trashAt)},
"is_trashed": ${stringify(isTrashed)},
"storage_classes_desired": ${JSON.stringify(storageClassesDesired, null, 2)},
"storage_classes_confirmed": ${JSON.stringify(storageClassesConfirmed, null, 2)},
"storage_classes_confirmed_at": ${stringify(storageClassesConfirmedAt)}`;

    return response;
};

const groupRequestApiResponse = (apiResponse: ProjectResource) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, description, groupClass, trashAt, isTrashed, deleteAt, properties } = apiResponse;
    const response = `"uuid": "${uuid}",
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
"properties": ${stringifyObject(properties)}`;

    return response;
};