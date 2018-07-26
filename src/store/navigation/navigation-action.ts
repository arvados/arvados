// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { projectActions, getProjectList } from "../project/project-action";
import { push } from "react-router-redux";
import { TreeItemStatus } from "../../components/tree/tree";
import { findTreeItem } from "../project/project-reducer";
import { RootState } from "../store";
import { Resource, ResourceKind } from "../../models/resource";
import { projectPanelActions } from "../project-panel/project-panel-action";

export const getResourceUrl = <T extends Resource>(resource: T): string => {
    switch (resource.kind) {
        case ResourceKind.PROJECT: return `/projects/${resource.uuid}`;
        case ResourceKind.COLLECTION: return `/collections/${resource.uuid}`;
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

            const resourceUrl = getResourceUrl(treeItem.data);

            if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
                if (router.location && !router.location.pathname.includes(resourceUrl)) {
                    dispatch(push(resourceUrl));
                }
                dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(treeItem.data.uuid));
            }

            const promise = treeItem.status === TreeItemStatus.LOADED
                ? Promise.resolve()
                : dispatch<any>(getProjectList(itemId));

            promise
                .then(() => dispatch<any>(() => {
                    if (itemMode === ItemMode.OPEN || itemMode === ItemMode.BOTH) {
                        dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(treeItem.data.uuid));
                    }
                    dispatch(projectPanelActions.RESET_PAGINATION());
                    dispatch(projectPanelActions.REQUEST_ITEMS());
                }));

        }
    };

