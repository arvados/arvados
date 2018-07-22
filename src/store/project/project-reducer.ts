// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

import { projectActions, ProjectAction } from "./project-action";
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { ProjectResource } from "../../models/project";

export type ProjectState = {
    items: Array<TreeItem<ProjectResource>>,
    currentItemId: string,
    creator: ProjectCreator
};

interface ProjectCreator {
    opened: boolean;
    pending: boolean;
    ownerUuid: string;
}

export function findTreeItem<T>(tree: Array<TreeItem<T>>, itemId: string): TreeItem<T> | undefined {
    let item;
    for (const t of tree) {
        item = t.id === itemId
            ? t
            : findTreeItem(t.items ? t.items : [], itemId);
        if (item) {
            break;
        }
    }
    return item;
}

export function getActiveTreeItem<T>(tree: Array<TreeItem<T>>): TreeItem<T> | undefined {
    let item;
    for (const t of tree) {
        item = t.active
            ? t
            : getActiveTreeItem(t.items ? t.items : []);
        if (item) {
            break;
        }
    }
    return item;
}

export function getTreePath<T>(tree: Array<TreeItem<T>>, itemId: string): Array<TreeItem<T>> {
    for (const item of tree) {
        if (item.id === itemId) {
            return [item];
        } else {
            const branch = getTreePath(item.items || [], itemId);
            if (branch.length > 0) {
                return [item, ...branch];
            }
        }
    }
    return [];
}

function resetTreeActivity<T>(tree: Array<TreeItem<T>>) {
    for (const t of tree) {
        t.active = false;
        resetTreeActivity(t.items ? t.items : []);
    }
}

function updateProjectTree(tree: Array<TreeItem<ProjectResource>>, projects: ProjectResource[], parentItemId?: string): Array<TreeItem<ProjectResource>> {
    let treeItem;
    if (parentItemId) {
        treeItem = findTreeItem(tree, parentItemId);
        if (treeItem) {
            treeItem.status = TreeItemStatus.Loaded;
        }
    }
    const items = projects.map(p => ({
        id: p.uuid,
        open: false,
        active: false,
        status: TreeItemStatus.Initial,
        data: p,
        items: []
    } as TreeItem<ProjectResource>));

    if (treeItem) {
        treeItem.items = items;
        return tree;
    }

    return items;
}

const updateCreator = (state: ProjectState, creator: Partial<ProjectCreator>) => ({
    ...state,
    creator: {
        ...state.creator,
        ...creator
    }
});

const initialState: ProjectState = {
    items: [],
    currentItemId: "",
    creator: {
        opened: false,
        pending: false,
        ownerUuid: ""
    }
};


export const projectsReducer = (state: ProjectState = initialState, action: ProjectAction) => {
    return projectActions.match(action, {
        OPEN_PROJECT_CREATOR: ({ ownerUuid }) => updateCreator(state, { ownerUuid, opened: true, pending: false }),
        CLOSE_PROJECT_CREATOR: () => updateCreator(state, { opened: false }),
        CREATE_PROJECT: () => updateCreator(state, { opened: false, pending: true }),
        CREATE_PROJECT_SUCCESS: () => updateCreator(state, { ownerUuid: "", pending: false }),
        CREATE_PROJECT_ERROR: () => updateCreator(state, { ownerUuid: "", pending: false }),
        REMOVE_PROJECT: () => state,
        PROJECTS_REQUEST: itemId => {
            const items = _.cloneDeep(state.items);
            const item = findTreeItem(items, itemId);
            if (item) {
                item.status = TreeItemStatus.Pending;
                state.items = items;
            }
            return { ...state, items };
        },
        PROJECTS_SUCCESS: ({ projects, parentItemId }) => {
            const items = _.cloneDeep(state.items);
            return {
                ...state,
                items: updateProjectTree(items, projects, parentItemId)
            };
        },
        TOGGLE_PROJECT_TREE_ITEM_OPEN: itemId => {
            const items = _.cloneDeep(state.items);
            const item = findTreeItem(items, itemId);
            if (item) {
                item.toggled = true;
                item.open = !item.open;
            }
            return {
                ...state,
                items,
                currentItemId: itemId
            };
        },
        TOGGLE_PROJECT_TREE_ITEM_ACTIVE: itemId => {
            const items = _.cloneDeep(state.items);
            resetTreeActivity(items);
            const item = findTreeItem(items, itemId);
            if (item) {
                item.toggled = true;
                item.active = true;
            }
            return {
                ...state,
                items,
                currentItemId: itemId
            };
        },
        RESET_PROJECT_TREE_ACTIVITY: () => {
            const items = _.cloneDeep(state.items);
            resetTreeActivity(items);
            return {
                ...state,
                items,
                currentItemId: ""
            };
        },
        default: () => state
    });
};
