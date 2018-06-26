// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import projectActions, { getProjectList } from "../project/project-action";
import { push } from "react-router-redux";
import { TreeItemStatus } from "../../components/tree/tree";
import { getCollectionList } from "../collection/collection-action";
import { findTreeItem } from "../project/project-reducer";
import { Resource, ResourceKind } from "../../models/resource";
import sidePanelActions from "../side-panel/side-panel-action";
import dataExplorerActions from "../data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "../../views/project-panel/project-panel";
import { projectPanelItems } from "../../views/project-panel/project-panel-selectors";
import { RootState } from "../store";

export const getResourceUrl = (resource: Resource): string => {
    switch (resource.kind) {
        case ResourceKind.LEVEL_UP: return `/projects/${resource.ownerUuid}`;
        case ResourceKind.PROJECT: return `/projects/${resource.uuid}`;
        case ResourceKind.COLLECTION: return `/collections/${resource.uuid}`;
        default:
            return "#";
    }
};

export enum ItemMode {
    BOTH,
    OPEN,
    ACTIVE
}

export const setProjectItem = (itemId: string, itemKind = ResourceKind.PROJECT, itemMode = ItemMode.OPEN) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { projects } = getState();

        let treeItem = findTreeItem(projects.items, itemId);
        if (treeItem && itemKind === ResourceKind.LEVEL_UP) {
            treeItem = findTreeItem(projects.items, treeItem.data.ownerUuid);
        }

        if (treeItem) {
            dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(treeItem.data.uuid));

            if (treeItem.status === TreeItemStatus.Loaded) {
                dispatch<any>(openProjectItem(treeItem.data, itemKind, itemMode));
            } else {
                dispatch<any>(getProjectList(itemId))
                    .then(() => dispatch<any>(openProjectItem(treeItem!.data, itemKind, itemMode)));
            }
            if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
                dispatch<any>(getCollectionList(itemId));
            }
        }
    };

const openProjectItem = (resource: Resource, itemKind: ResourceKind, itemMode: ItemMode) =>
    (dispatch: Dispatch, getState: () => RootState) => {

        const { collections, projects } = getState();

        if (itemMode === ItemMode.OPEN || itemMode === ItemMode.BOTH) {
            dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(resource.uuid));
        }

        if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
            dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(resource.uuid));
        }

        dispatch(push(getResourceUrl({ ...resource, kind: itemKind })));
        dispatch(dataExplorerActions.SET_ITEMS({
            id: PROJECT_PANEL_ID,
            items: projectPanelItems(
                projects.items,
                resource.uuid,
                collections
            )
        }));
    };
