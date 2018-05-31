// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getType } from "typesafe-actions";
import { Project } from "../models/project";
import { actions, ProjectAction } from "./project-action";
import { TreeItem } from "../components/tree/tree";
import * as _ from 'lodash';

type ProjectState = Array<TreeItem<Project>>;

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

const projectsReducer = (state: ProjectState = [], action: ProjectAction) => {
    switch (action.type) {
        case getType(actions.createProject): {
            return [...state, action.payload];
        }
        case getType(actions.toggleProjectTreeItem): {
            const tree = _.cloneDeep(state);
            const itemId = action.payload;
            const item = findTreeItem(tree, itemId);
            if (item) {
                item.open = !item.open;
            }
            return tree;
        }
        default:
            return state;
    }
};

export default projectsReducer;
