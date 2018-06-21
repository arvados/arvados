// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

import { Project } from "../../models/project";
import actions, { ProjectAction } from "./project-action";
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";

export type ProjectState = Array<TreeItem<Project>>;

function findTreeItem<T>(tree: Array<TreeItem<T>>, itemId: string): TreeItem<T> | undefined {
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
    const items = projects.map((p, idx) => ({
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

const projectsReducer = (state: ProjectState = [], action: ProjectAction) => {
    return actions.match(action, {
        CREATE_PROJECT: project => [...state, project],
        REMOVE_PROJECT: () => state,
        PROJECTS_REQUEST: itemId => {
            const tree = _.cloneDeep(state);
            const item = findTreeItem(tree, itemId);
            if (item) {
                item.status = TreeItemStatus.Pending;
            }
            return tree;
        },
        PROJECTS_SUCCESS: ({ projects, parentItemId }) => {
            return updateProjectTree(state, projects, parentItemId);
        },
        TOGGLE_PROJECT_TREE_ITEM_OPEN: itemId => {
            const tree = _.cloneDeep(state);
            const item = findTreeItem(tree, itemId);
            if (item) {
                item.toggled = true;
                item.open = !item.open;
            }
            return tree;
        },
        TOGGLE_PROJECT_TREE_ITEM_ACTIVE: itemId => {
            const tree = _.cloneDeep(state);
            resetTreeActivity(tree);
            const item = findTreeItem(tree, itemId);
            if (item) {
                item.active = true;
            }
            return tree;
        },
        RESET_PROJECT_TREE_ACTIVITY: () => {
            const tree = _.cloneDeep(state);
            resetTreeActivity(tree);
            return tree;
        },
        default: () => state
    });
};

export default projectsReducer;
