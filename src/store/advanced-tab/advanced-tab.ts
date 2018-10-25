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

export const ADVANCED_TAB_DIALOG = 'advancedTabDialog';

export interface AdvancedTabDialogData {
    kind: string;
    pythonHeader: string;
    pythonExample: string;
    CLIGetHeader: string;
    CLIGetExample: string;
    CLIUpdateHeader: string;
    CLIUpdateExample: string;
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
    (dispatch: Dispatch<any>, getState: () => RootState) => {
        const { resources } = getState();
        const kind = extractUuidKind(uuid);
        const data = getResource(uuid)(resources);
        if (data) {
            if (kind === ResourceKind.COLLECTION) {
                const dataCollection: AdvancedTabDialogData = {
                    kind,
                    pythonHeader: pythonHeader(CollectionData.COLLECTION),
                    pythonExample: pythonExample(data.uuid, GroupContentsResourcePrefix.COLLECTION),
                    CLIGetHeader: CLIGetHeader(CollectionData.COLLECTION),
                    CLIGetExample: CLIGetExample(data.uuid, GroupContentsResourcePrefix.COLLECTION),
                    CLIUpdateHeader: CLIUpdateHeader(CollectionData.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED),
                    CLIUpdateExample: CLIUpdateExample(data.uuid, CollectionData.COLLECTION, data.storageClassesConfirmed, CollectionData.STORAGE_CLASSES_CONFIRMED),
                    curlHeader: curlHeader(CollectionData.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED),
                    curlExample: curlExample(data.uuid, GroupContentsResourcePrefix.COLLECTION, data.storageClassesConfirmed, CollectionData.COLLECTION, CollectionData.STORAGE_CLASSES_CONFIRMED)
                };
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataCollection }));
            } else if (kind === ResourceKind.PROCESS) {
                const dataProcess: AdvancedTabDialogData = {
                    kind,
                    pythonHeader: pythonHeader(ProcessData.CONTAINER_REQUEST),
                    pythonExample: pythonExample(data.uuid, GroupContentsResourcePrefix.PROCESS),
                    CLIGetHeader: CLIGetHeader(ProcessData.CONTAINER_REQUEST),
                    CLIGetExample: CLIGetExample(data.uuid, GroupContentsResourcePrefix.PROCESS),
                    CLIUpdateHeader: CLIUpdateHeader(ProcessData.CONTAINER_REQUEST, ProcessData.OUTPUT_NAME),
                    CLIUpdateExample: CLIUpdateExample(data.uuid, ProcessData.CONTAINER_REQUEST, data.outputName, ProcessData.OUTPUT_NAME),
                    curlHeader: curlHeader(ProcessData.CONTAINER_REQUEST, ProcessData.OUTPUT_NAME),
                    curlExample: curlExample(data.uuid, GroupContentsResourcePrefix.PROCESS, data.outputName, ProcessData.CONTAINER_REQUEST, ProcessData.OUTPUT_NAME)
                };
                dispatch(dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data: dataProcess }));
            } else if (kind === ResourceKind.PROJECT) {
                const dataProject: AdvancedTabDialogData = {
                    kind,
                    pythonHeader: pythonHeader(ProjectData.GROUP),
                    pythonExample: pythonExample(data.uuid, GroupContentsResourcePrefix.PROJECT),
                    CLIGetHeader: CLIGetHeader(ProjectData.GROUP),
                    CLIGetExample: CLIGetExample(data.uuid, GroupContentsResourcePrefix.PROJECT),
                    CLIUpdateHeader: CLIUpdateHeader(ProjectData.GROUP, ProjectData.DELETE_AT),
                    CLIUpdateExample: CLIUpdateExample(data.uuid, ProjectData.GROUP, data.deleteAt, ProjectData.DELETE_AT),
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

const CLIGetHeader = (resourceKind: string) =>
    `An example arv command to get a ${resourceKind} using its uuid:`;

const CLIGetExample = (uuid: string, resourcePrefix: string) => {
    const cliGetExample = `arv ${resourcePrefix} get \\
 --uuid ${uuid}`;

    return cliGetExample;
};

const CLIUpdateHeader = (resourceKind: string, resourceName: string) =>
    `An example arv command to update the "${resourceName}" attribute for the current ${resourceKind}:`;

const CLIUpdateExample = (uuid: string, resourceKind: string, resource: string | string[], resourceName: string) => {
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