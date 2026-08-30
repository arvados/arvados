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
import { UserResource } from 'models/user';
import { LinkResource } from 'models/link';
import { WorkflowResource } from 'models/workflow';
import { KeepServiceResource } from 'models/keep-services';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import { ExternalCredential } from 'models/external-credential';
import pick from 'lodash/pick';
import snakeCase from 'lodash/snakeCase';

export const ADVANCED_TAB_DIALOG = 'advancedTabDialog';

export interface AdvancedTabDialogData {
    uuid: string;
    apiResponse: string;
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
    LINKS = 'links',
    EXTERNAL_CREDENTIALS = 'external_credentials',
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
    EXPIRES_AT = 'expires_at'
}

enum LinkData {
    LINK = 'link',
    PROPERTIES = 'properties'
}

enum WorkflowData {
    WORKFLOW = 'workflow',
    CREATED_AT = 'created_at'
}

enum ExternalCredentialData {
    EXTERNAL_CREDENTIAL = 'external_credential',
    CREATED_AT = 'created_at'
}

type AdvanceResourceKind = CollectionData | ProcessData | ProjectData | RepositoryData | SshKeyData | VirtualMachineData | KeepServiceData | ApiClientAuthorizationsData | UserData | LinkData | WorkflowData | ExternalCredentialData;
type AdvanceResourcePrefix = GroupContentsResourcePrefix | ResourcePrefix;
type AdvanceResponseData = ContainerRequestResource | ProjectResource | CollectionResource | RepositoryResource | SshKeyResource | VirtualMachinesResource | KeepServiceResource | ApiClientAuthorization | UserResource | LinkResource | WorkflowResource | ExternalCredential | undefined;

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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'createdAt', 'modifiedByUserUuid', 'modifiedAt', 'portableDataHash',
                        'replicationDesired', 'replicationConfirmedAt', 'replicationConfirmed',
                        'name', 'description', 'properties', 'deleteAt', 'trashAt', 'isTrashed',
                        'storageClassesDesired', 'storageClassesConfirmed', 'storageClassesConfirmedAt',
                        'currentVersionUuid', 'version', 'preserveVersion', 'fileCount', 'fileSizeTotal'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'createdAt', 'modifiedAt', 'modifiedByUserUuid',
                        'name', 'description', 'properties', 'state', 'requestingContainerUuid', 'containerUuid',
                        'containerCountMax', 'mounts', 'runtimeConstraints', 'containerImage', 'environment', 'cwd', 'command', 'outputPath', 'priority', 'expiresAt', 'filters', 'containerCount',
                        'useExisting', 'schedulingParameters', 'outputUuid', 'logUuid', 'outputName', 'outputTtl', 'outputGlob'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'createdAt', 'modifiedByUserUuid', 'modifiedAt',
                        'name', 'description', 'groupClass', 'trashAt', 'isTrashed', 'deleteAt', 'properties',
                        'canWrite', 'canManage'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'modifiedByUserUuid', 'modifiedAt', 'name', 'createdAt', 'cloneUrls'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'authorizedUserUuid', 'modifiedByUserUuid', 'modifiedAt', 'name', 'createdAt', 'expiresAt'
                    ],
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
                    apiResponseKeys: [
                        'hostname', 'uuid', 'ownerUuid', 'modifiedByUserUuid', 'modifiedAt', 'createdAt'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'modifiedByUserUuid', 'modifiedAt', 'serviceHost', 'servicePort', 'serviceSslFlag', 'serviceType', 'createdAt', 'readOnly'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'createdAt', 'modifiedByUserUuid', 'modifiedAt',
                        'email', 'firstName', 'lastName', 'username', 'isActive', 'isAdmin', 'prefs'
                    ],
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
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'apiToken', 'createdByIpAddress', 'lastUsedByIpAddress',
                        'lastUsedAt', 'expiresAt', 'createdAt', 'updatedAt', 'scopes'
                    ],
                    data: dataApiClientAuthorization,
                    resourceKind: ApiClientAuthorizationsData.API_CLIENT_AUTHORIZATION,
                    resourcePrefix: ResourcePrefix.API_CLIENT_AUTHORIZATIONS,
                    resourceKindProperty: ApiClientAuthorizationsData.EXPIRES_AT,
                    property: dataApiClientAuthorization!.createdAt
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
                    apiResponseKeys: [
                        'uuid', 'name', 'headUuid', 'headKind', 'tailUuid', 'tailKind', 'linkClass',
                        'ownerUuid', 'createdAt', 'modifiedAt', 'modifiedByUserUuid', 'properties'
                    ],
                    data: dataLink,
                    resourceKind: LinkData.LINK,
                    resourcePrefix: ResourcePrefix.LINKS,
                    resourceKindProperty: LinkData.PROPERTIES,
                    property: dataLink!.properties
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataLink));
                break;
            case ResourceKind.WORKFLOW:
                const wfResources = getState().resources;
                const dataWf = getResource<WorkflowResource>(uuid)(wfResources);
                const advanceDataWf = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKeys: [
                        'uuid', 'name', 'ownerUuid', 'createdAt', 'modifiedAt', 'modifiedByUserUuid', 'description'
                    ],
                    data: dataWf,
                    resourceKind: WorkflowData.WORKFLOW,
                    resourcePrefix: GroupContentsResourcePrefix.WORKFLOW,
                    resourceKindProperty: WorkflowData.CREATED_AT,
                    property: dataWf!.createdAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataWf));
                break;
            case ResourceKind.EXTERNAL_CREDENTIAL:
                const { resources: ecResources } = getState();
                const dataExtCred = getResource<ExternalCredential>(uuid)(ecResources);
                const advanceDataExtCred = advancedTabData({
                    uuid,
                    metadata: '',
                    user: '',
                    apiResponseKeys: [
                        'uuid', 'ownerUuid', 'createdAt', 'modifiedByUserUuid', 'modifiedAt', 'name', 'description', 'scopes', 'expiresAt'
                    ],
                    data: dataExtCred,
                    resourceKind: ExternalCredentialData.EXTERNAL_CREDENTIAL,
                    resourcePrefix: ResourcePrefix.EXTERNAL_CREDENTIALS,
                    resourceKindProperty: ExternalCredentialData.CREATED_AT,
                    property: dataExtCred!.createdAt
                });
                dispatch<any>(initAdvancedTabDialog(advanceDataExtCred));
                break;

            default:
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Could not open advanced tab for this resource.", hideDuration: 10000, kind: SnackbarKind.ERROR }));
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

        return { data, metadata };
    };

const initAdvancedTabDialog = (data: AdvancedTabDialogData) => dialogActions.OPEN_DIALOG({ id: ADVANCED_TAB_DIALOG, data });

interface AdvancedTabData {
    uuid: string;
    metadata: ListResults<LinkResource> | string;
    user: UserResource | string;
    apiResponseKeys: string[];
    data: AdvanceResponseData;
    resourceKind: AdvanceResourceKind;
    resourcePrefix: AdvanceResourcePrefix;
    resourceKindProperty: AdvanceResourceKind;
    property: any;
}

const advancedTabData = ({ uuid, user, metadata, apiResponseKeys, data, resourceKind, resourcePrefix, resourceKindProperty, property }: AdvancedTabData) => {
    return {
        uuid,
        user,
        metadata,
        apiResponse: formatApiResponse(data, apiResponseKeys),
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
  -H "Authorization: Bearer $ARVADOS_API_TOKEN" \\
  --data-urlencode ${resourceKind}@/dev/stdin \\
  https://$ARVADOS_API_HOST/arvados/v1/${resourcePrefix}/${uuid} \\
  <<EOF
{
  "${resourceName}": ${JSON.stringify(resource, null, 4)}
}
EOF`;

    return curlExample;
};

const formatApiResponse = (apiResponse: any, keys: string[]): string => {
    const picked = pick(apiResponse, keys);
    const snaked: Record<string, any> = {};
    for (const key in picked) {
        snaked[snakeCase(key)] = picked[key];
    }
    return JSON.stringify(snaked, null, 2);
};

