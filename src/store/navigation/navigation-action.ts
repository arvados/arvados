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

export const setProjectItem = (projects: Array<TreeItem<Project>>, itemId: string, itemKind: ResourceKind) => (dispatch: Dispatch) => {

    const openProjectItem = (resource: Resource) => {
        dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(resource.uuid));
        dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(resource.uuid));
        dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(resource.uuid));
        dispatch(push(getResourceUrl({...resource, kind: itemKind})));
    };
    const treeItem = findTreeItem(projects, itemId);

    if (treeItem) {
        if (treeItem.status === TreeItemStatus.Loaded) {
            openProjectItem(treeItem.data);
        } else {
            dispatch<any>(getProjectList(itemId))
                .then(() => openProjectItem(treeItem.data));
        }
        dispatch<any>(getCollectionList(itemId));

        // if (item.type === ResourceKind.PROJECT || item.type === ResourceKind.LEVEL_UP) {
        //     this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(item.uuid));
        // }
        // this.props.dispatch<any>(getCollectionList(item.uuid));

    }
};

    // toggleProjectTreeItemActive = (itemId: string, status: TreeItemStatus) => {
    //     if (status === TreeItemStatus.Loaded) {
    //         this.openProjectItem(itemId);
    //         this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
    //         this.props.dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(itemId));
    //     } else {
    //         this.props.dispatch<any>(getProjectList(itemId))
    //             .then(() => {
    //                 this.openProjectItem(itemId);
    //                 this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
    //                 this.props.dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(itemId));
    //             });
    //     }
    // }
