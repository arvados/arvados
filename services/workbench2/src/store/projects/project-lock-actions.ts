// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ServiceRepository } from "services/services";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { loadResource } from "store/resources/resources-actions";
import { RootState } from "store/store";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { addDisabledButton, removeDisabledButton } from "store/multiselect/multiselect-actions";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

export const freezeProject = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(addDisabledButton(ContextMenuActionNames.FREEZE_PROJECT))
    const userUUID = getState().auth.user!.uuid;
    let updatedProject;

    try {
        updatedProject = await services.projectService.update(uuid, {
            frozenByUuid: userUUID,
        });
    } catch (e) {
        console.error(e);
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not freeze project', hideDuration: 4000, kind: SnackbarKind.ERROR }));
    }

    dispatch(projectPanelDataActions.REQUEST_ITEMS());
    dispatch<any>(loadResource(uuid, false));
    dispatch<any>(removeDisabledButton(ContextMenuActionNames.FREEZE_PROJECT))
    return updatedProject;
};


export const unfreezeProject = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(addDisabledButton(ContextMenuActionNames.FREEZE_PROJECT))
    const updatedProject = await services.projectService.update(uuid, {
        frozenByUuid: null,
    });

    dispatch(projectPanelDataActions.REQUEST_ITEMS());
    dispatch<any>(loadResource(uuid, false));
    dispatch<any>(removeDisabledButton(ContextMenuActionNames.FREEZE_PROJECT))
    return updatedProject;
};
