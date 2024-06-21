// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { ContextMenuPosition } from "./context-menu-reducer";
import { ContextMenuKind } from "views-components/context-menu/menu-item-sort";
import { Dispatch } from "redux";
import { RootState } from "store/store";
import { getResource, getResourceWithEditableStatus } from "../resources/resources";
import { UserResource } from "models/user";
import { isSidePanelTreeCategory } from "store/side-panel-tree/side-panel-tree-actions";
import { extractUuidKind, ResourceKind, EditableResource, Resource } from "models/resource";
import { Process, isProcessCancelable } from "store/processes/process";
import { RepositoryResource } from "models/repositories";
import { SshKeyResource } from "models/ssh-key";
import { VirtualMachinesResource } from "models/virtual-machines";
import { KeepServiceResource } from "models/keep-services";
import { ProcessResource } from "models/process";
import { CollectionResource } from "models/collection";
import { GroupClass, GroupResource } from "models/group";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { LinkResource } from "models/link";
import { resourceIsFrozen } from "common/frozen-resources";
import { ProjectResource } from "models/project";
import { getProcess } from "store/processes/process";
import { filterCollectionFilesBySelection } from "store/collection-panel/collection-panel-files/collection-panel-files-state";

export const contextMenuActions = unionize({
    OPEN_CONTEXT_MENU: ofType<{ position: ContextMenuPosition; resource: ContextMenuResource }>(),
    CLOSE_CONTEXT_MENU: ofType<{}>(),
});

export type ContextMenuAction = UnionOf<typeof contextMenuActions>;

export type ContextMenuResource = {
    name: string;
    uuid: string;
    ownerUuid: string;
    description?: string;
    kind: ResourceKind;
    menuKind: ContextMenuKind | string;
    isTrashed?: boolean;
    isEditable?: boolean;
    outputUuid?: string;
    workflowUuid?: string;
    isAdmin?: boolean;
    isFrozen?: boolean;
    storageClassesDesired?: string[];
    properties?: { [key: string]: string | string[] };
    isMulti?: boolean;
    fromContextMenu?: boolean;
};

export const isKeyboardClick = (event: React.MouseEvent<HTMLElement>) => event.nativeEvent.detail === 0;

export const openContextMenu = (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource) => (dispatch: Dispatch) => {
    event.preventDefault();
    const { left, top } = event.currentTarget.getBoundingClientRect();
    dispatch(
        contextMenuActions.OPEN_CONTEXT_MENU({
            position: {
                x: event.clientX || left,
                y: event.clientY || top,
            },
            resource,
        })
    );
};

export const openCollectionFilesContextMenu =
    (event: React.MouseEvent<HTMLElement>, isWritable: boolean) => (dispatch: Dispatch, getState: () => RootState) => {
        const selectedCount = filterCollectionFilesBySelection(getState().collectionPanelFiles, true).length;
        const multiple = selectedCount > 1;
        dispatch<any>(
            openContextMenu(event, {
                name: "",
                uuid: "",
                ownerUuid: "",
                description: "",
                kind: ResourceKind.COLLECTION,
                menuKind:
                    selectedCount > 0
                        ? isWritable
                            ? multiple
                                ? ContextMenuKind.COLLECTION_FILES_MULTIPLE
                                : ContextMenuKind.COLLECTION_FILES
                            : multiple
                            ? ContextMenuKind.READONLY_COLLECTION_FILES_MULTIPLE
                            : ContextMenuKind.READONLY_COLLECTION_FILES
                        : ContextMenuKind.COLLECTION_FILES_NOT_SELECTED,
            })
        );
    };

export const openRepositoryContextMenu =
    (event: React.MouseEvent<HTMLElement>, repository: RepositoryResource) => (dispatch: Dispatch, getState: () => RootState) => {
        dispatch<any>(
            openContextMenu(event, {
                name: "",
                uuid: repository.uuid,
                ownerUuid: repository.ownerUuid,
                kind: ResourceKind.REPOSITORY,
                menuKind: ContextMenuKind.REPOSITORY,
            })
        );
    };

export const openVirtualMachinesContextMenu =
    (event: React.MouseEvent<HTMLElement>, repository: VirtualMachinesResource) => (dispatch: Dispatch, getState: () => RootState) => {
        dispatch<any>(
            openContextMenu(event, {
                name: "",
                uuid: repository.uuid,
                ownerUuid: repository.ownerUuid,
                kind: ResourceKind.VIRTUAL_MACHINE,
                menuKind: ContextMenuKind.VIRTUAL_MACHINE,
            })
        );
    };

export const openSshKeyContextMenu = (event: React.MouseEvent<HTMLElement>, sshKey: SshKeyResource) => (dispatch: Dispatch) => {
    dispatch<any>(
        openContextMenu(event, {
            name: "",
            uuid: sshKey.uuid,
            ownerUuid: sshKey.ownerUuid,
            kind: ResourceKind.SSH_KEY,
            menuKind: ContextMenuKind.SSH_KEY,
        })
    );
};

export const openKeepServiceContextMenu = (event: React.MouseEvent<HTMLElement>, keepService: KeepServiceResource) => (dispatch: Dispatch) => {
    dispatch<any>(
        openContextMenu(event, {
            name: "",
            uuid: keepService.uuid,
            ownerUuid: keepService.ownerUuid,
            kind: ResourceKind.KEEP_SERVICE,
            menuKind: ContextMenuKind.KEEP_SERVICE,
        })
    );
};

export const openApiClientAuthorizationContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => (dispatch: Dispatch) => {
    dispatch<any>(
        openContextMenu(event, {
            name: "",
            uuid: resourceUuid,
            ownerUuid: "",
            kind: ResourceKind.API_CLIENT_AUTHORIZATION,
            menuKind: ContextMenuKind.API_CLIENT_AUTHORIZATION,
        })
    );
};

export const openRootProjectContextMenu =
    (event: React.MouseEvent<HTMLElement>, projectUuid: string) => (dispatch: Dispatch, getState: () => RootState) => {
        const res = getResource<UserResource>(projectUuid)(getState().resources);
        if (res) {
            dispatch<any>(
                openContextMenu(event, {
                    name: "",
                    uuid: res.uuid,
                    ownerUuid: res.uuid,
                    kind: res.kind,
                    menuKind: ContextMenuKind.ROOT_PROJECT,
                    isTrashed: false,
                })
            );
        }
    };

export const openProjectContextMenu =
    (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => (dispatch: Dispatch, getState: () => RootState) => {
        const res = getResource<GroupContentsResource>(resourceUuid)(getState().resources);
        const menuKind = dispatch<any>(resourceUuidToContextMenuKind(resourceUuid));
        if (res && menuKind) {
            dispatch<any>(
                openContextMenu(event, {
                    name: res.name,
                    uuid: res.uuid,
                    kind: res.kind,
                    menuKind,
                    description: res.description,
                    ownerUuid: res.ownerUuid,
                    isTrashed: "isTrashed" in res ? res.isTrashed : false,
                    isFrozen: !!(res as ProjectResource).frozenByUuid,
                })
            );
        }
    };

export const openSidePanelContextMenu = (event: React.MouseEvent<HTMLElement>, id: string) => (dispatch: Dispatch, getState: () => RootState) => {
    if (!isSidePanelTreeCategory(id)) {
        const kind = extractUuidKind(id);
        if (kind === ResourceKind.USER) {
            dispatch<any>(openRootProjectContextMenu(event, id));
        } else if (kind === ResourceKind.PROJECT) {
            dispatch<any>(openProjectContextMenu(event, id));
        }
    }
};

export const openProcessContextMenu = (event: React.MouseEvent<HTMLElement>, process: Process) => (dispatch: Dispatch, getState: () => RootState) => {
    const res = getResource<ProcessResource>(process.containerRequest.uuid)(getState().resources);
    if (res) {
        dispatch<any>(
            openContextMenu(event, {
                uuid: res.uuid,
                ownerUuid: res.ownerUuid,
                kind: ResourceKind.PROCESS,
                name: res.name,
                description: res.description,
                outputUuid: res.outputUuid || "",
                workflowUuid: res.properties.template_uuid || "",
                menuKind: isProcessCancelable(process) ? ContextMenuKind.RUNNING_PROCESS_RESOURCE : ContextMenuKind.PROCESS_RESOURCE
            })
        );
    }
};

export const openPermissionEditContextMenu =
    (event: React.MouseEvent<HTMLElement>, link: LinkResource) => (dispatch: Dispatch, getState: () => RootState) => {
        if (link) {
            dispatch<any>(
                openContextMenu(event, {
                    name: link.name,
                    uuid: link.uuid,
                    kind: link.kind,
                    menuKind: ContextMenuKind.PERMISSION_EDIT,
                    ownerUuid: link.ownerUuid,
                })
            );
        }
    };

export const openUserContextMenu = (event: React.MouseEvent<HTMLElement>, user: UserResource) => (dispatch: Dispatch, getState: () => RootState) => {
    dispatch<any>(
        openContextMenu(event, {
            name: "",
            uuid: user.uuid,
            ownerUuid: user.ownerUuid,
            kind: user.kind,
            menuKind: ContextMenuKind.USER,
        })
    );
};

export const resourceUuidToContextMenuKind =
    (uuid: string, readonly = false) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { isAdmin: isAdminUser, uuid: userUuid } = getState().auth.user!;
        const kind = extractUuidKind(uuid);
        const resource = getResourceWithEditableStatus<GroupResource & EditableResource>(uuid, userUuid)(getState().resources);
        const isFrozen = resourceIsFrozen(resource, getState().resources);
        const isEditable = (isAdminUser || (resource || ({} as EditableResource)).isEditable) && !readonly && !isFrozen;

        switch (kind) {
            case ResourceKind.PROJECT:
                if (isFrozen) {
                    return isAdminUser ? ContextMenuKind.FROZEN_PROJECT_ADMIN : ContextMenuKind.FROZEN_PROJECT;
                }

                return isAdminUser && !readonly
                    ? resource && resource.groupClass !== GroupClass.FILTER
                        ? ContextMenuKind.PROJECT_ADMIN
                        : ContextMenuKind.FILTER_GROUP_ADMIN
                    : isEditable
                    ? resource && resource.groupClass !== GroupClass.FILTER
                        ? ContextMenuKind.PROJECT
                        : ContextMenuKind.FILTER_GROUP
                    : ContextMenuKind.READONLY_PROJECT;
            case ResourceKind.COLLECTION:
                const c = getResource<CollectionResource>(uuid)(getState().resources);
                if (c === undefined) {
                    return;
                }
                const isOldVersion = c.uuid !== c.currentVersionUuid;
                const isTrashed = c.isTrashed;
                return isOldVersion
                    ? ContextMenuKind.OLD_VERSION_COLLECTION
                    : isTrashed && isEditable
                    ? ContextMenuKind.TRASHED_COLLECTION
                    : isAdminUser && isEditable
                    ? ContextMenuKind.COLLECTION_ADMIN
                    : isEditable
                    ? ContextMenuKind.COLLECTION
                    : ContextMenuKind.READONLY_COLLECTION;
            case ResourceKind.PROCESS:
                return isAdminUser && isEditable
                    ? resource && isProcessCancelable(getProcess(resource.uuid)(getState().resources) as Process)
                        ? ContextMenuKind.RUNNING_PROCESS_ADMIN
                        : ContextMenuKind.PROCESS_ADMIN
                    : readonly
                    ? ContextMenuKind.READONLY_PROCESS_RESOURCE
                    : resource && isProcessCancelable(getProcess(resource.uuid)(getState().resources) as Process)
                    ? ContextMenuKind.RUNNING_PROCESS_RESOURCE
                    : ContextMenuKind.PROCESS_RESOURCE;
            case ResourceKind.USER:
                return ContextMenuKind.ROOT_PROJECT;
            case ResourceKind.LINK:
                return ContextMenuKind.LINK;
            case ResourceKind.WORKFLOW:
                return isEditable ? ContextMenuKind.WORKFLOW : ContextMenuKind.READONLY_WORKFLOW;
            default:
                return;
        }
    };

export const openSearchResultsContextMenu =
    (event: React.MouseEvent<HTMLElement>, uuid: string) => (dispatch: Dispatch, getState: () => RootState) => {
        const res = getResource<Resource>(uuid)(getState().resources);
        if (res) {
            dispatch<any>(
                openContextMenu(event, {
                    name: "",
                    uuid: res.uuid,
                    ownerUuid: "",
                    kind: res.kind,
                    menuKind: ContextMenuKind.SEARCH_RESULTS,
                })
            );
        }
    };
