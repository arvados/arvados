// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import projectActions, { getProjectList } from "../project/project-action";
import { push } from "react-router-redux";
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { getCollectionList } from "../collection/collection-action";
import { findTreeItem } from "../project/project-reducer";
import { Project } from "../../models/project";
import { Resource, ResourceKind } from "../../models/resource";
import sidePanelActions from "../side-panel/side-panel-action";

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

export const setProjectItem = (projects: Array<TreeItem<Project>>, itemId: string, itemKind: ResourceKind, itemMode: ItemMode) => (dispatch: Dispatch) => {

    const openProjectItem = (resource: Resource) => {
        if (itemMode === ItemMode.OPEN || itemMode === ItemMode.BOTH) {
            dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(resource.uuid));
        }

        if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
            dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(resource.uuid));
        }

        dispatch(push(getResourceUrl({...resource, kind: itemKind})));
    };

    let treeItem = findTreeItem(projects, itemId);
    if (treeItem && itemKind === ResourceKind.LEVEL_UP) {
        treeItem = findTreeItem(projects, treeItem.data.ownerUuid);
    }

    if (treeItem) {
        dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(treeItem.data.uuid));

        if (treeItem.status === TreeItemStatus.Loaded) {
            openProjectItem(treeItem.data);
        } else {
            dispatch<any>(getProjectList(itemId))
                .then(() => openProjectItem(treeItem!.data));
        }
        if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
            dispatch<any>(getCollectionList(itemId));
        }
    }
};
