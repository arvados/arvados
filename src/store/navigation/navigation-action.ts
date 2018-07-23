// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { projectActions, getProjectList } from "../project/project-action";
import { push } from "react-router-redux";
import { TreeItemStatus } from "../../components/tree/tree";
import { findTreeItem } from "../project/project-reducer";
import { dataExplorerActions } from "../data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "../../views/project-panel/project-panel";
import { RootState } from "../store";
import { Resource, ResourceKind } from "../../models/resource";

export const getResourceUrl = <T extends Resource>(resource: T): string => {
    switch (resource.kind) {
        case ResourceKind.Project: return `/projects/${resource.uuid}`;
        case ResourceKind.Collection: return `/collections/${resource.uuid}`;
        default: return resource.href;
    }
};

export enum ItemMode {
    BOTH,
    OPEN,
    ACTIVE
}

export const setProjectItem = (itemId: string, itemMode: ItemMode) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { projects, router } = getState();
        const treeItem = findTreeItem(projects.items, itemId);

        if (treeItem) {

            if (itemMode === ItemMode.OPEN || itemMode === ItemMode.BOTH) {
                dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(treeItem.data.uuid));
            }

            const resourceUrl = getResourceUrl(treeItem.data);

            if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
                if (router.location && !router.location.pathname.includes(resourceUrl)) {
                    dispatch(push(resourceUrl));
                }
                dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(treeItem.data.uuid));
            }

            const promise = treeItem.status === TreeItemStatus.Loaded
                ? Promise.resolve()
                : dispatch<any>(getProjectList(itemId));

            promise
                .then(() => dispatch<any>(() => {
                    dispatch(dataExplorerActions.RESET_PAGINATION({id: PROJECT_PANEL_ID}));
                    dispatch(dataExplorerActions.REQUEST_ITEMS({id: PROJECT_PANEL_ID}));
                }));

        }
    };
