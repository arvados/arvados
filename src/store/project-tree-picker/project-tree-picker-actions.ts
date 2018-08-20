// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { TreePickerId, receiveTreePickerData } from "~/views-components/project-tree-picker/project-tree-picker";
import { mockProjectResource } from "~/models/test-utils";
import { treePickerActions } from "~/store/tree-picker/tree-picker-actions";

export const resetPickerProjectTree = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(treePickerActions.RESET_TREE_PICKER({pickerId: TreePickerId.PROJECTS}));
    dispatch<any>(treePickerActions.RESET_TREE_PICKER({pickerId: TreePickerId.SHARED_WITH_ME}));
    dispatch<any>(treePickerActions.RESET_TREE_PICKER({pickerId: TreePickerId.FAVORITES}));

    dispatch<any>(initPickerProjectTree());
};

export const initPickerProjectTree = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const uuid = services.authService.getUuid();

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
    return receiveTreePickerData(
        '',
        [mockProjectResource({ uuid, name: kind })],
        kind
    );
};