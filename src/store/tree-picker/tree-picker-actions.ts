// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";

import { TreePickerNode } from "./tree-picker";
import { receiveTreePickerData, TreePickerKind } from "../../views-components/project-tree-picker/project-tree-picker";
import { mockProjectResource } from "../../models/test-utils";
import { Dispatch } from "redux";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";

export const treePickerActions = unionize({
    LOAD_TREE_PICKER_NODE: ofType<{ id: string, pickerId: string }>(),
    LOAD_TREE_PICKER_NODE_SUCCESS: ofType<{ id: string, nodes: Array<TreePickerNode>, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_COLLAPSE: ofType<{ id: string, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_SELECT: ofType<{ id: string, pickerId: string }>()
}, {
        tag: 'type',
        value: 'payload'
    });

export const initPickerProjectTree = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const uuid = services.authService.getUuid();

    dispatch<any>(getPickerTreeProjects(uuid));
    dispatch<any>(getSharedWithMeProjectsPickerTree(uuid));
    dispatch<any>(getFavoritesProjectsPickerTree(uuid));
};

const getPickerTreeProjects = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerKind.PROJECTS);
};

const getSharedWithMeProjectsPickerTree = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerKind.SHARED_WITH_ME);
};

const getFavoritesProjectsPickerTree = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerKind.FAVORITES);
};

const getProjectsPickerTree = (uuid: string, kind: string) => {
    return receiveTreePickerData(
        '',
        [mockProjectResource({ uuid, name: kind })],
        kind
    );
};

export type TreePickerAction = UnionOf<typeof treePickerActions>;
