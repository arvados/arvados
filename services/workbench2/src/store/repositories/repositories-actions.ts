// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from "services/services";
import { navigateToRepositories } from "store/navigation/navigation-action";
import { unionize, ofType, UnionOf } from "common/unionize";
import { dialogActions } from 'store/dialog/dialog-actions';
import { RepositoryResource } from "models/repositories";
import { startSubmit, reset, stopSubmit, FormErrors } from "redux-form";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

export const repositoriesActions = unionize({
    SET_REPOSITORIES: ofType<any>(),
});

export type RepositoriesActions = UnionOf<typeof repositoriesActions>;

export const REPOSITORIES_PANEL = 'repositoriesPanel';
export const REPOSITORIES_SAMPLE_GIT_DIALOG = 'repositoriesSampleGitDialog';
export const REPOSITORY_ATTRIBUTES_DIALOG = 'repositoryAttributesDialog';
export const REPOSITORY_CREATE_FORM_NAME = 'repositoryCreateFormName';
export const REPOSITORY_REMOVE_DIALOG = 'repositoryRemoveDialog';

export const openRepositoriesSampleGitDialog = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuidPrefix = getState().properties.uuidPrefix;
        dispatch(dialogActions.OPEN_DIALOG({ id: REPOSITORIES_SAMPLE_GIT_DIALOG, data: { uuidPrefix } }));
    };

export const openRepositoryAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const repositoryData = getState().repositories.items.find(it => it.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: REPOSITORY_ATTRIBUTES_DIALOG, data: { repositoryData } }));
    };

export const openRepositoryCreateDialog = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        const user = await services.userService.get(userUuid!);
        dispatch(reset(REPOSITORY_CREATE_FORM_NAME));
        dispatch(dialogActions.OPEN_DIALOG({ id: REPOSITORY_CREATE_FORM_NAME, data: { user } }));
    };

export const createRepository = (repository: RepositoryResource) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        const user = await services.userService.get(userUuid!);
        dispatch(startSubmit(REPOSITORY_CREATE_FORM_NAME));
        try {
            const newRepository = await services.repositoriesService.create({ name: `${user.username}/${repository.name}` });
            dispatch(dialogActions.CLOSE_DIALOG({ id: REPOSITORY_CREATE_FORM_NAME }));
            dispatch(reset(REPOSITORY_CREATE_FORM_NAME));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Repository has been successfully created.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            dispatch<any>(loadRepositoriesData());
            return newRepository;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.NAME_HAS_ALREADY_BEEN_TAKEN) {
                dispatch(stopSubmit(REPOSITORY_CREATE_FORM_NAME, { name: 'Repository with the same name already exists.' } as FormErrors));
            }
            return undefined;
        }
    };

export const openRemoveRepositoryDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: REPOSITORY_REMOVE_DIALOG,
            data: {
                title: 'Remove repository',
                text: 'Are you sure you want to remove this repository?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeRepository = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.repositoriesService.delete(uuid);
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        dispatch<any>(loadRepositoriesData());
    };

const repositoriesBindedActions = bindDataExplorerActions(REPOSITORIES_PANEL);

export const openRepositoriesPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
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
