// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { ContextMenuPosition } from "./context-menu-reducer";
import { ContextMenuKind } from "views-components/context-menu/menu-item-sort";
import { Dispatch } from "redux";
import { RootState } from "store/store";
import { getResource } from "../resources/resources";
import { UserResource } from "models/user";
import { isSidePanelTreeCategory } from "store/side-panel-tree/side-panel-tree-actions";
import { extractUuidKind, ResourceKind, Resource } from "models/resource";
import { RepositoryResource } from "models/repositories";
import { SshKeyResource } from "models/ssh-key";
import { VirtualMachinesResource } from "models/virtual-machines";
import { KeepServiceResource } from "models/keep-services";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { LinkResource } from "models/link";
import { ProjectResource } from "models/project";
import { Process } from "store/processes/process";
import { filterCollectionFilesBySelection } from "store/collection-panel/collection-panel-files/collection-panel-files-state";
import { selectOne, deselectAllOthers } from "store/multiselect/multiselect-actions";
import { ContainerRequestResource } from "models/container-request";
import { resourceToMenuKind } from "common/resource-to-menu-kind";

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

export const openContextMenuAndSelect = (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource) => (dispatch: Dispatch) => {
    event.preventDefault();
    dispatch<any>(selectOne(resource.uuid));
    dispatch<any>(deselectAllOthers(resource.uuid));
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

const openContextMenuOnly = (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource) => (dispatch: Dispatch) => {
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
            openContextMenuAndSelect(event, {
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
            openContextMenuAndSelect(event, {
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
            openContextMenuAndSelect(event, {
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
        openContextMenuAndSelect(event, {
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
        openContextMenuAndSelect(event, {
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
        openContextMenuAndSelect(event, {
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
                openContextMenuOnly(event, {
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
        const menuKind = dispatch<any>(resourceToMenuKind(resourceUuid));
        if (res && menuKind) {
            dispatch<any>(
                openContextMenuOnly(event, {
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
    const res = getResource<ContainerRequestResource>(process.containerRequest.uuid)(getState().resources);
    const menuKind = dispatch<any>(resourceToMenuKind(process.containerRequest.uuid));
    if (res && menuKind) {
        dispatch<any>(
            openContextMenuAndSelect(event, {
                uuid: process.containerRequest.uuid,
                ownerUuid: process.containerRequest.ownerUuid,
                kind: menuKind,
                name: process.containerRequest.name,
                description: process.containerRequest.description,
                outputUuid: process.containerRequest.outputUuid || "",
                workflowUuid: process.containerRequest.properties.template_uuid || "",
                menuKind
            })
        );
    }
};

export const openPermissionEditContextMenu =
    (event: React.MouseEvent<HTMLElement>, link: LinkResource) => (dispatch: Dispatch, getState: () => RootState) => {
        if (link) {
            dispatch<any>(
                openContextMenuAndSelect(event, {
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
        openContextMenuAndSelect(event, {
            name: "",
            uuid: user.uuid,
            ownerUuid: user.ownerUuid,
            kind: user.kind,
            menuKind: ContextMenuKind.USER,
        })
    );
};

export const openSearchResultsContextMenu =
    (event: React.MouseEvent<HTMLElement>, uuid: string) => (dispatch: Dispatch, getState: () => RootState) => {
        const res = getResource<Resource>(uuid)(getState().resources);
        if (res) {
            dispatch<any>(
                openContextMenuAndSelect(event, {
                    name: "",
                    uuid: res.uuid,
                    ownerUuid: "",
                    kind: res.kind,
                    menuKind: ContextMenuKind.SEARCH_RESULTS,
                })
            );
        }
    };
