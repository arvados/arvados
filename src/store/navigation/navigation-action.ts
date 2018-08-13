// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { projectActions, getProjectList } from "../project/project-action";
import { push } from "react-router-redux";
import { TreeItemStatus } from "~/components/tree/tree";
import { findTreeItem } from "../project/project-reducer";
import { RootState } from "../store";
import { Resource, ResourceKind } from "~/models/resource";
import { projectPanelActions } from "../project-panel/project-panel-action";
import { getCollectionUrl } from "~/models/collection";
import { getProjectUrl, ProjectResource } from "~/models/project";
import { ProjectService } from "~/services/project-service/project-service";
import { ServiceRepository } from "~/services/services";
import { sidePanelActions } from "../side-panel/side-panel-action";
import { SidePanelIdentifiers } from "../side-panel/side-panel-reducer";
import { getUuidObjectType, ObjectTypes } from "~/models/object-types";

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
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
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
        } else {
            const uuid = services.authService.getUuid();
            if (itemId === uuid) {
                dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(uuid));
                dispatch(projectPanelActions.RESET_PAGINATION());
                dispatch(projectPanelActions.REQUEST_ITEMS());
            }
        }
    };

export const restoreBranch = (itemId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = await loadProjectAncestors(itemId, services.projectService);
        const uuids = ancestors.map(ancestor => ancestor.uuid);
        await loadBranch(uuids, dispatch);
        dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(SidePanelIdentifiers.PROJECTS));
        dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(SidePanelIdentifiers.PROJECTS));
        uuids.forEach(uuid => {
            dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(uuid));
        });
    };

export const loadProjectAncestors = async (uuid: string, projectService: ProjectService): Promise<Array<ProjectResource>> => {
    if (getUuidObjectType(uuid) === ObjectTypes.USER) {
        return [];
    } else {
        const currentProject = await projectService.get(uuid);
        const ancestors = await loadProjectAncestors(currentProject.ownerUuid, projectService);
        return [...ancestors, currentProject];
    }
};

const loadBranch = async (uuids: string[], dispatch: Dispatch): Promise<any> => {
    const [uuid, ...rest] = uuids;
    if (uuid) {
        await dispatch<any>(getProjectList(uuid));
        return loadBranch(rest, dispatch);
    }
};
