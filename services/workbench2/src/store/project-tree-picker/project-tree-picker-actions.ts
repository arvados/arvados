// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from "services/services";
import { mockProjectResource } from "models/test-utils";
import { treePickerActions, receiveTreePickerProjectsData } from "store/tree-picker/tree-picker-actions";
import { TreePickerId } from 'models/tree';

export const resetPickerProjectTree = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(treePickerActions.RESET_TREE_PICKER({ pickerId: TreePickerId.PROJECTS }));
    dispatch<any>(treePickerActions.RESET_TREE_PICKER({ pickerId: TreePickerId.SHARED_WITH_ME }));
    dispatch<any>(treePickerActions.RESET_TREE_PICKER({ pickerId: TreePickerId.FAVORITES }));

    dispatch<any>(initPickerProjectTree());
};

export const initPickerProjectTree = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const uuid = getUserUuid(getState());
    if (!uuid) { return; }
    dispatch<any>(getPickerTreeProjects(uuid));
    dispatch<any>(getSharedWithMeProjectsPickerTree(uuid));
    dispatch<any>(getFavoritesProjectsPickerTree(uuid));
};

const getPickerTreeProjects = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerId.PROJECTS);
};

const getSharedWithMeProjectsPickerTree = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerId.SHARED_WITH_ME);
};

const getFavoritesProjectsPickerTree = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerId.FAVORITES);
};

const getProjectsPickerTree = (uuid: string, kind: string) => {
    return receiveTreePickerProjectsData(
        '',
        [mockProjectResource({ uuid, name: kind })],
        kind
    );
};
