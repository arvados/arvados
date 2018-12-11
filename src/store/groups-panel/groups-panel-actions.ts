// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { reset } from 'redux-form';
import { bindDataExplorerActions } from "~/store/data-explorer/data-explorer-action";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { Person } from '~/views-components/sharing-dialog/people-select';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { getResource } from '~/store/resources/resources';
import { GroupResource } from '~/models/group';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';

export const GROUPS_PANEL_ID = "groupsPanel";
export const CREATE_GROUP_DIALOG = "createGroupDialog";
export const CREATE_GROUP_FORM = "createGroupForm";
export const CREATE_GROUP_NAME_FIELD_NAME = 'name';
export const CREATE_GROUP_USERS_FIELD_NAME = 'users';
export const GROUP_ATTRIBUTES_DIALOG = 'groupAttributesDialog';
export const GROUP_REMOVE_DIALOG = 'groupRemoveDialog';

export const GroupsPanelActions = bindDataExplorerActions(GROUPS_PANEL_ID);

export const loadGroupsPanel = () => GroupsPanelActions.REQUEST_ITEMS();

export const openCreateGroupDialog = () =>
    (dispatch: Dispatch) => {
        dispatch(dialogActions.OPEN_DIALOG({ id: CREATE_GROUP_DIALOG, data: {} }));
        dispatch(reset(CREATE_GROUP_FORM));
    };

export const openGroupAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<GroupResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: GROUP_ATTRIBUTES_DIALOG, data }));
    };

export const removeGroup = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
        await services.groupsService.delete(uuid);
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        dispatch<any>(loadGroupsPanel());
    };

export const openRemoveGroupDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: GROUP_REMOVE_DIALOG,
            data: {
                title: 'Remove group',
                text: 'Are you sure you want to remove this group?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export interface CreateGroupFormData {
    [CREATE_GROUP_NAME_FIELD_NAME]: string;
    [CREATE_GROUP_USERS_FIELD_NAME]: Person[];
}

export const createGroup = (data: CreateGroupFormData) =>
    (dispatch: Dispatch) => {
        console.log(data);
    };
