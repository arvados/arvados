// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from '~/store/dialog/dialog-actions';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { ResourceKind, extractUuidKind, Resource } from '~/models/resource';
import { getResource } from '~/store/resources/resources';
import { GroupContentsResourcePrefix } from '~/services/groups-service/groups-service';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { ContainerRequestResource } from '~/models/container-request';
import { CollectionResource } from '~/models/collection';

export const ADVANCED_TAB_DIALOG = 'advancedTabDialog';

export interface AdvancedTabDialogData {
    apiResponse: any;
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

interface AdvancedData extends Resource {
    storageClassesConfirmed: string[];
    outputName: string;
    deleteAt: string;
}

export const openAdvancedTabDialog = (uuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const { resources } = getState();
        const kind = extractUuidKind(uuid);
        const data = getResource<any>(uuid)(resources);
        if (data) {
            console.log(data);
            if (kind === ResourceKind.COLLECTION) {
                const dataCollection: AdvancedTabDialogData = {
                    apiResponse: collectionApiResponse(data),
                    pythonHeader: pythonHeader(CollectionData.COLLECTION),
                    pythonExample: pythonExample(data.uuid, GroupContentsResourcePrefix.COLLECTION),
                    cliGetHeader: cliGetHeader(CollectionData.COLLECTION),
                    cliGetExample: cliGetExample(data.uuid, GroupContentsResourcePrefix.COLLECTION),
                    cliUpdateHeader: cliUpdateHeader(CollectionData.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED),
                    cliUpdateExample: cliUpdateExample(data.uuid, CollectionData.COLLECTION, data.storageClassesConfirmed, CollectionData.STORAGE_CLASSES_CONFIRMED),
                    curlHeader: curlHeader(CollectionData.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED),
                    curlExample: curlExample(data.uuid, GroupContentsResourcePrefix.COLLECTION, data.storageClassesConfirmed, CollectionData.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED)
                };
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataCollection }));
            } else if (kind === ResourceKind.PROCESS) {
                const dataProcess: AdvancedTabDialogData = {
                    apiResponse: containerRequestApiResponse(data),
                    pythonHeader: pythonHeader(ProcessData.CONTAINER_REQUEST),
                    pythonExample: pythonExample(data.uuid, GroupContentsResourcePrefix.PROCESS),
                    cliGetHeader: cliGetHeader(ProcessData.CONTAINER_REQUEST),
                    cliGetExample: cliGetExample(data.uuid, GroupContentsResourcePrefix.PROCESS),
                    cliUpdateHeader: cliUpdateHeader(ProcessData.CONTAINER_REQUEST, ProcessData.OUTPUT_NAME),
                    cliUpdateExample: cliUpdateExample(data.uuid, ProcessData.CONTAINER_REQUEST, data.outputName, ProcessData.OUTPUT_NAME),
                    curlHeader: curlHeader(ProcessData.CONTAINER_REQUEST, ProcessData.OUTPUT_NAME),
                    curlExample: curlExample(data.uuid, GroupContentsResourcePrefix.PROCESS, data.outputName, ProcessData.CONTAINER_REQUEST, ProcessData.OUTPUT_NAME)
                };
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataProcess }));
            } else if (kind === ResourceKind.PROJECT) {
                const dataProject: AdvancedTabDialogData = {
                    apiResponse: `'${data}'`,
                    pythonHeader: pythonHeader(ProjectData.GROUP),
                    pythonExample: pythonExample(data.uuid, GroupContentsResourcePrefix.PROJECT),
                    cliGetHeader: cliGetHeader(ProjectData.GROUP),
                    cliGetExample: cliGetExample(data.uuid, GroupContentsResourcePrefix.PROJECT),
                    cliUpdateHeader: cliUpdateHeader(ProjectData.GROUP, ProjectData.DELETE_AT),
                    cliUpdateExample: cliUpdateExample(data.uuid, ProjectData.GROUP, data.deleteAt, ProjectData.DELETE_AT),
                    curlHeader: curlHeader(ProjectData.GROUP, ProjectData.DELETE_AT),
                    curlExample: curlExample(data.uuid, GroupContentsResourcePrefix.PROJECT, data.deleteAt, ProjectData.GROUP, ProjectData.DELETE_AT)
                };
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataProject }));
            }
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Could not open advanced tab for this resource.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
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

const containerRequestApiResponse = (apiResponse: ContainerRequestResource) => {
    const response = `{
    "uuid": "${apiResponse.uuid}",
    "owner_uuid": "${apiResponse.ownerUuid}",
    "created_at": "${apiResponse.createdAt}",
    "modified_at": "${apiResponse.modifiedAt}",
    "modified_by_client_uuid": "${apiResponse.modifiedByClientUuid}",
    "modified_by_user_uuid": "${apiResponse.modifiedByUserUuid}",
    "name": "${apiResponse.name}",
    "description": "${apiResponse.description}",
    "properties": "${apiResponse.properties}",
    "state": "${apiResponse.state}",
    "requesting_container_uuid": "${apiResponse.requestingContainerUuid}",
    "container_uuid": "${apiResponse.containerUuid}",
    "container_count_max": "${apiResponse.containerCountMax}",
    "mounts": "${apiResponse.mounts}",
    "runtime_constraints": "${apiResponse.runtimeConstraints}",
    "container_image": "${apiResponse.containerImage}",
    "environment": "${apiResponse.environment}",
    "cwd": "${apiResponse.cwd}",
    "command": "${apiResponse.command}",
    "output_path": "${apiResponse.outputPath}",
    "priority": "${apiResponse.priority}",
    "expires_at": "${apiResponse.expiresAt}",
    "filters": "${apiResponse.filters}"
    "use_existing": "${apiResponse.useExisting}",
    "output_uuid": "${apiResponse.outputUuid}",
    "scheduling_parameters": "${apiResponse.schedulingParameters}",
    "kind": "${apiResponse.kind}",
    "log_uuid": "${apiResponse.logUuid}",
    "output_name": "${apiResponse.outputName}",
    "output_ttl": "${apiResponse.outputTtl}",
}`;

    return response;
};

const collectionApiResponse = (apiResponse: CollectionResource) => {
    const response = `{
    "uuid": "${apiResponse.uuid}",
    "owner_uuid": "${apiResponse.ownerUuid}",
    "created_at": "${apiResponse.createdAt}",
    "modified_at": "${apiResponse.modifiedAt}",
    "modified_by_client_uuid": "${apiResponse.modifiedByClientUuid}",
    "modified_by_user_uuid": "${apiResponse.modifiedByUserUuid}",
    "portable_data_hash": "${apiResponse.portableDataHash}",
    "replication_desired": "${apiResponse.replicationDesired}",
    "replication_confirmed_at": "${apiResponse.replicationConfirmedAt}",
    "replication_confirmed": "${apiResponse.replicationConfirmed}",

    "manifest_text": "${apiResponse.manifestText}",
    "name": "${apiResponse.name}",
    "description": "${apiResponse.description}",
    "properties": "${apiResponse.properties}",
    "delete_at": "${apiResponse.deleteAt}",
    
    "trash_at": "${apiResponse.trashAt}",
    "is_trashed": "${apiResponse.isTrashed}",
    

    
}`;

    return response;
};

const groupRequestApiResponse = (apiResponse: ContainerRequestResource) => {
    const response = `{
    "uuid": "${apiResponse.uuid}",
    "owner_uuid": "${apiResponse.ownerUuid}",
    "created_at": "${apiResponse.createdAt}",
    "modified_at": "${apiResponse.modifiedAt}",
    "modified_by_client_uuid": "${apiResponse.modifiedByClientUuid}",
    "modified_by_user_uuid": "${apiResponse.modifiedByUserUuid}",
    "name": "${apiResponse.name}",
    "description": "${apiResponse.description}",
    "properties": "${apiResponse.properties}",
    "state": "${apiResponse.state}",
    "requesting_container_uuid": "${apiResponse.requestingContainerUuid}",
    "container_uuid": "${apiResponse.containerUuid}",
    "container_count_max": "${apiResponse.containerCountMax}",
    "mounts": "${apiResponse.mounts}",
    "runtime_constraints": "${apiResponse.runtimeConstraints}",
    "container_image": "${apiResponse.containerImage}",
    "environment": "${apiResponse.environment}",
    "cwd": "${apiResponse.cwd}",
    "command": "${apiResponse.command}",
    "output_path": "${apiResponse.outputPath}",
    "priority": "${apiResponse.priority}",
    "expires_at": "${apiResponse.expiresAt}",
    "filters": "${apiResponse.filters}"
    "use_existing": "${apiResponse.useExisting}",
    "output_uuid": "${apiResponse.outputUuid}",
    "scheduling_parameters": "${apiResponse.schedulingParameters}",
    "kind": "${apiResponse.kind}",
    "log_uuid": "${apiResponse.logUuid}",
    "output_name": "${apiResponse.outputName}",
    "output_ttl": "${apiResponse.outputTtl}",
}`;

    return response;
};