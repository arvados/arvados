// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { ContextMenuPosition, ContextMenuResource } from "./context-menu-reducer";
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { getResource } from '../resources/resources';
import { ProjectResource } from '~/models/project';
import { UserResource } from '~/models/user';
import { isSidePanelTreeCategory } from '~/store/side-panel-tree/side-panel-tree-actions';
import { extractUuidKind, ResourceKind } from '~/models/resource';

export const contextMenuActions = unionize({
    OPEN_CONTEXT_MENU: ofType<{ position: ContextMenuPosition, resource: ContextMenuResource }>(),
    CLOSE_CONTEXT_MENU: ofType<{}>()
});

export type ContextMenuAction = UnionOf<typeof contextMenuActions>;

export type ContextMenuResource = {
    name: string;
    uuid: string;
    ownerUuid: string;
    description?: string;
    kind: ContextMenuKind;
    isTrashed?: boolean;
}

export const openContextMenu = (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource) =>
    (dispatch: Dispatch) => {
        event.preventDefault();
        dispatch(
            contextMenuActions.OPEN_CONTEXT_MENU({
                position: { x: event.clientX, y: event.clientY },
                resource
            })
        );
    };

export const openRootProjectContextMenu = (event: React.MouseEvent<HTMLElement>, projectUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const res = getResource<UserResource>(projectUuid)(getState().resources);
        if (res) {
            dispatch<any>(openContextMenu(event, {
                name: '',
                uuid: res.uuid,
                ownerUuid: res.uuid,
                kind: ContextMenuKind.ROOT_PROJECT,
                isTrashed: false
            }));
        }
    };

export const openProjectContextMenu = (event: React.MouseEvent<HTMLElement>, projectUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const res = getResource<ProjectResource>(projectUuid)(getState().resources);
        if (res) {
            dispatch<any>(openContextMenu(event, {
                name: res.name,
                uuid: res.uuid,
                kind: ContextMenuKind.PROJECT,
                ownerUuid: res.ownerUuid,
                isTrashed: res.isTrashed
            }));
        }
    };

export const openSidePanelContextMenu = (event: React.MouseEvent<HTMLElement>, id: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        if (!isSidePanelTreeCategory(id)) {
            const kind = extractUuidKind(id);
            if (kind === ResourceKind.USER) {
                dispatch<any>(openRootProjectContextMenu(event, id));
            } else if (kind === ResourceKind.PROJECT) {
                dispatch<any>(openProjectContextMenu(event, id));
            }
        }
    };

export const openProcessContextMenu = (event: React.MouseEvent<HTMLElement>) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const resource = {
            uuid: '',
            name: '',
            description: '',
            kind: ContextMenuKind.PROCESS
        };
        dispatch<any>(openContextMenu(event, resource));
    };

export const resourceKindToContextMenuKind = (uuid: string) => {
    const kind = extractUuidKind(uuid);
    switch (kind) {
        case ResourceKind.PROJECT:
            return ContextMenuKind.PROJECT;
        case ResourceKind.COLLECTION:
            return ContextMenuKind.COLLECTION_RESOURCE;
        case ResourceKind.USER:
            return ContextMenuKind.ROOT_PROJECT;
        default:
            return;
    }
};
