// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

import { Project } from "../../models/project";
import actions, { ProjectAction } from "./project-action";
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";

export type ProjectState = {
    items: Array<TreeItem<Project>>,
    currentItemId: string
};

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
    for (const item of tree){
        if(item.id === itemId){
            return [item];
        } else {
            const branch = getTreePath(item.items || [], itemId);
            if(branch.length > 0){
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

function updateProjectTree(tree: Array<TreeItem<Project>>, projects: Project[], parentItemId?: string): Array<TreeItem<Project>> {
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
    } as TreeItem<Project>));

    if (treeItem) {
        treeItem.items = items;
        return tree;
    }

    return items;
}

const projectsReducer = (state: ProjectState = { items: [], currentItemId: "" }, action: ProjectAction) => {
    return actions.match(action, {
        CREATE_PROJECT: project => ({
            ...state,
            items: state.items.concat({
                id: project.uuid,
                open: false,
                active: false,
                status: TreeItemStatus.Loaded,
                toggled: false,
                items: [],
                data: project
            })
        }),
        REMOVE_PROJECT: () => state,
        PROJECTS_REQUEST: itemId => {
            const items = _.cloneDeep(state.items);
            const item = findTreeItem(items, itemId);
            if (item) {
                item.status = TreeItemStatus.Pending;
                state.items = items;
            }
            return state;
        },
        PROJECTS_SUCCESS: ({ projects, parentItemId }) => {
            return {
                ...state,
                items: updateProjectTree(state.items, projects, parentItemId)
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
                items
            };
        },
        TOGGLE_PROJECT_TREE_ITEM_ACTIVE: itemId => {
            const items = _.cloneDeep(state.items);
            resetTreeActivity(items);
            const item = findTreeItem(items, itemId);
            if (item) {
                item.active = true;
            }
            return {
                items,
                currentItemId: itemId
            };
        },
        RESET_PROJECT_TREE_ACTIVITY: () => {
            const items = _.cloneDeep(state.items);
            resetTreeActivity(items);
            return {
                ...state,
                items
            };
        },
        default: () => state
    });
};

export default projectsReducer;
