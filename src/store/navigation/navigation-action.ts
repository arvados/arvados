// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { projectActions, getProjectList } from "../project/project-action";
import { push } from "react-router-redux";
import { TreeItemStatus } from "../../components/tree/tree";
import { findTreeItem, getTreePath } from "../project/project-reducer";
import { dataExplorerActions } from "../data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "../../views/project-panel/project-panel";
import { RootState } from "../store";
import { Resource, ResourceKind } from "../../models/resource";
import { getCollectionUrl } from "../../models/collection";
import { getProjectUrl, ProjectResource } from "../../models/project";
import { projectService } from "../../services/services";

export const getResourceUrl = <T extends Resource>(resource: T): string => {
    switch (resource.kind) {
        case ResourceKind.PROJECT: return getProjectUrl(resource.uuid);
        case ResourceKind.COLLECTION: return getCollectionUrl(resource.uuid);
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
            console.log('treeItem', treeItem);

            const treePath = getTreePath(projects.items, treeItem.data.uuid);

            console.log('treePath', treePath);
            const resourceUrl = getResourceUrl(treeItem.data);

            console.log('resourceUrl', resourceUrl);
            const ancestors = loadProjectAncestors(treeItem.data.uuid);
            console.log('ancestors', ancestors);

            if (itemMode === ItemMode.ACTIVE || itemMode === ItemMode.BOTH) {
                if (router.location && !router.location.pathname.includes(resourceUrl)) {
                    dispatch(push(resourceUrl));
                }
                dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(treePath[treePath.length - 1].id));
            }

            const promise = treeItem.status === TreeItemStatus.LOADED
                ? Promise.resolve()
                : dispatch<any>(getProjectList(itemId));

            promise
                .then(() => dispatch<any>(() => {
                    if (itemMode === ItemMode.OPEN || itemMode === ItemMode.BOTH) {
                        dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(treeItem.data.uuid));
                    }
                    dispatch(dataExplorerActions.RESET_PAGINATION({id: PROJECT_PANEL_ID}));
                    dispatch(dataExplorerActions.REQUEST_ITEMS({id: PROJECT_PANEL_ID}));
                }));

        }
    };

    const USER_UUID_REGEX = /.*tpzed.*/;

    export const loadProjectAncestors = async (uuid: string): Promise<Array<ProjectResource>> => {
        if (USER_UUID_REGEX.test(uuid)) {
            return [];
        } else {
            const currentProject = await projectService.get(uuid);
            const ancestors = await loadProjectAncestors(currentProject.ownerUuid);
            return [...ancestors, currentProject];
        }
    };