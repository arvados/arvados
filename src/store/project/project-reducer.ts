// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { projectActions, ProjectAction } from "./project-action";
import { TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { ProjectResource } from "~/models/project";

export type ProjectState = {
    items: Array<TreeItem<ProjectResource>>,
    currentItemId: string,
    creator: ProjectCreator,
    updater: ProjectUpdater
};

interface ProjectCreator {
    opened: boolean;
    ownerUuid: string;
    error?: string;
}

interface ProjectUpdater {
    opened: boolean;
    uuid: string;
}

function rebuildTree<T>(tree: Array<TreeItem<T>>, action: (item: TreeItem<T>, visitedItems: TreeItem<T>[]) => void, visitedItems: TreeItem<T>[] = []): Array<TreeItem<T>> {
    const newTree: Array<TreeItem<T>> = [];
    for (const t of tree) {
        const items = t.items
            ? rebuildTree(t.items, action, visitedItems.concat(t))
            : undefined;
        const item: TreeItem<T> = { ...t, items };
        action(item, visitedItems);
        newTree.push(item);
    }
    return newTree;
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

const updateCreator = (state: ProjectState, creator: Partial<ProjectCreator>) => ({
    ...state,
    creator: {
        ...state.creator,
        ...creator
    }
});

const updateProject = (state: ProjectState, updater?: Partial<ProjectUpdater>) => ({
    ...state,
    updater: {
        ...state.updater,
        ...updater
    }
});

const initialState: ProjectState = {
    items: [],
    currentItemId: "",
    creator: {
        opened: false,
        ownerUuid: ""
    },
    updater: {
        opened: false,
        uuid: ''
    }
};

export const projectsReducer = (state: ProjectState = initialState, action: ProjectAction) => {
    return projectActions.match(action, {
        OPEN_PROJECT_CREATOR: ({ ownerUuid }) => updateCreator(state, { ownerUuid, opened: true }),
        CLOSE_PROJECT_CREATOR: () => updateCreator(state, { opened: false }),
        CREATE_PROJECT: () => updateCreator(state, { error: undefined }),
        CREATE_PROJECT_SUCCESS: () => updateCreator(state, { opened: false, ownerUuid: "" }),
        OPEN_PROJECT_UPDATER: ({ uuid }) => updateProject(state, { uuid, opened: true }),
        CLOSE_PROJECT_UPDATER: () => updateProject(state, { opened: false, uuid: "" }),
        UPDATE_PROJECT_SUCCESS: () => updateProject(state, { opened: false, uuid: "" }),
        REMOVE_PROJECT: () => state,
        PROJECTS_REQUEST: itemId => {
            return {
                ...state,
                items: rebuildTree(state.items, item => {
                    if (item.id === itemId) {
                        item.status = TreeItemStatus.PENDING;
                    }
                })
            };
        },
        PROJECTS_SUCCESS: ({ projects, parentItemId }) => {
            const items = projects.map(p => ({
               id: p.uuid,
               open: false,
               active: false,
               status: TreeItemStatus.INITIAL,
               data: p,
               items: []
            }));
            return {
                ...state,
                items: state.items.length > 0 ?
                    rebuildTree(state.items, item => {
                        if (item.id === parentItemId) {
                           item.status = TreeItemStatus.LOADED;
                           item.items = items;
                        }
                    }) : items
            };
        },
        TOGGLE_PROJECT_TREE_ITEM_OPEN: ({ itemId, open, recursive }) => ({
            ...state,
            items: rebuildTree(state.items, (item, visitedItems) => {
                if (item.id === itemId) {
                    if (recursive && open !== undefined) {
                        visitedItems.forEach(item => item.open = open);
                    }
                    item.open = open !== undefined ? open : !item.open;
                }
            }),
            currentItemId: itemId
        }),
        TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ({ itemId, active, recursive }) => ({
            ...state,
            items: rebuildTree(state.items, (item, visitedItems) => {
                item.active = false;
                if (item.id === itemId) {
                    if (recursive && active !== undefined) {
                        visitedItems.forEach(item => item.active = active);
                    }

                    item.active = active !== undefined ? active : true;
                }
            }),
            currentItemId: itemId
        }),
        RESET_PROJECT_TREE_ACTIVITY: () => ({
            ...state,
            items: rebuildTree(state.items, item => {
                item.active = false;
            }),
            currentItemId: ""
        }),
        default: () => state
    });
};
