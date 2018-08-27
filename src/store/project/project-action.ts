// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { ProjectResource } from "~/models/project";
import { Dispatch } from "redux";
import { FilterBuilder } from "~/common/api/filter-builder";
import { RootState } from "../store";
import { updateFavorites } from "../favorites/favorites-actions";
import { ServiceRepository } from "~/services/services";
import { resourcesActions } from '~/store/resources/resources-actions';

export const projectActions = unionize({
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<string>(),
    PROJECTS_SUCCESS: ofType<{ projects: ProjectResource[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<string>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<string>(),
    RESET_PROJECT_TREE_ACTIVITY: ofType<string>()
});

export const getProjectList = (parentUuid: string = '') => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(projectActions.PROJECTS_REQUEST(parentUuid));
        return services.projectService.list({
            filters: new FilterBuilder()
                .addEqual("ownerUuid", parentUuid)
                .getFilters()
        }).then(({ items: projects }) => {
            dispatch(projectActions.PROJECTS_SUCCESS({ projects, parentItemId: parentUuid }));
            dispatch<any>(updateFavorites(projects.map(project => project.uuid)));
            dispatch<any>(resourcesActions.SET_RESOURCES(projects));
            return projects;
        });
    };

export type ProjectAction = UnionOf<typeof projectActions>;
