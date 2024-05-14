// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ServiceRepository } from "services/services";
import { projectPanelActions } from "store/project-panel/project-panel-action-bind";
import { loadResource } from "store/resources/resources-actions";
import { RootState } from "store/store";
import { MultiSelectMenuActionNames } from "components/multiselect-toolbar/ms-menu-actions"; 
import { addDisabledButton, removeDisabledButton } from "store/multiselect/multiselect-actions";

export const freezeProject = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(addDisabledButton(MultiSelectMenuActionNames.FREEZE_PROJECT))
    const userUUID = getState().auth.user!.uuid;
    
    const updatedProject = await services.projectService.update(uuid, {
        frozenByUuid: userUUID,
    });
    
    dispatch(projectPanelActions.REQUEST_ITEMS());
    dispatch<any>(loadResource(uuid, false));
    dispatch<any>(removeDisabledButton(MultiSelectMenuActionNames.FREEZE_PROJECT))
    return updatedProject;
};

export const unfreezeProject = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(addDisabledButton(MultiSelectMenuActionNames.FREEZE_PROJECT))
    const updatedProject = await services.projectService.update(uuid, {
        frozenByUuid: null,
    });

    dispatch(projectPanelActions.REQUEST_ITEMS());
    dispatch<any>(loadResource(uuid, false));
    dispatch<any>(removeDisabledButton(MultiSelectMenuActionNames.FREEZE_PROJECT))
    return updatedProject;
};
