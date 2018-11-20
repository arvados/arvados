// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { RootState } from '~/store/store';
import { ServiceRepository } from "~/services/services";
import { navigateToRepositories } from "~/store/navigation/navigation-action";
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { dialogActions } from '~/store/dialog/dialog-actions';

export const repositoriesActions = unionize({
    SET_REPOSITORIES: ofType<any>(),
});

export type RepositoriesActions = UnionOf<typeof repositoriesActions>;

export const REPOSITORIES_PANEL = 'repositoriesPanel';
export const REPOSITORIES_SAMPLE_GIT_NAME = 'repositoriesSampleGit';

export const openRepositoriesSampleGitDialog = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuidPrefix = getState().properties.uuidPrefix;
        dispatch(dialogActions.OPEN_DIALOG({ id: REPOSITORIES_SAMPLE_GIT_NAME, data: { uuidPrefix } }));
    };

const repositoriesBindedActions = bindDataExplorerActions(REPOSITORIES_PANEL);

export const openRepositoriesPanel = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateToRepositories);
    };

export const loadRepositoriesData = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const repositories = await services.repositoriesService.list();
        dispatch(repositoriesActions.SET_REPOSITORIES(repositories.items));
    };

export const loadRepositoriesPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(repositoriesBindedActions.REQUEST_ITEMS());
    };