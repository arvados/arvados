// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ServiceRepository } from "services/services";
import { projectPanelActions } from "store/project-panel/project-panel-action";
import { loadResource } from "store/resources/resources-actions";
import { RootState } from "store/store";

export const freezeProject = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUUID = getState().auth.user!.uuid;

        const updatedProject = await services.projectService.update(uuid, {
            frozenByUuid: userUUID
        });

        dispatch(projectPanelActions.REQUEST_ITEMS());
        dispatch<any>(loadResource(uuid, false));
        return updatedProject;
    };

export const unfreezeProject = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {

        const updatedProject = await services.projectService.update(uuid, {
            frozenByUuid: null
        });

        dispatch(projectPanelActions.REQUEST_ITEMS());
        dispatch<any>(loadResource(uuid, false));
        return updatedProject;
    };