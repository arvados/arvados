// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { reset, startSubmit, stopSubmit, FormErrors } from 'redux-form';
import { bindDataExplorerActions } from "~/store/data-explorer/data-explorer-action";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { Person } from '~/views-components/sharing-dialog/people-select';
import { ServiceRepository } from '~/services/services';
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/services/common-service/common-resource-service';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';

export const GROUPS_PANEL_ID = "groupsPanel";
export const CREATE_GROUP_DIALOG = "createGroupDialog";
export const CREATE_GROUP_FORM = "createGroupForm";
export const CREATE_GROUP_NAME_FIELD_NAME = 'name';
export const CREATE_GROUP_USERS_FIELD_NAME = 'users';

export const GroupsPanelActions = bindDataExplorerActions(GROUPS_PANEL_ID);

export const loadGroupsPanel = () => GroupsPanelActions.REQUEST_ITEMS();

export const openCreateGroupDialog = () =>
    (dispatch: Dispatch) => {
        dispatch(dialogActions.OPEN_DIALOG({ id: CREATE_GROUP_DIALOG, data: {} }));
        dispatch(reset(CREATE_GROUP_FORM));
    };

export interface CreateGroupFormData {
    [CREATE_GROUP_NAME_FIELD_NAME]: string;
    [CREATE_GROUP_USERS_FIELD_NAME]?: Person[];
}

export const createGroup = ({ name }: CreateGroupFormData) =>
    async (dispatch: Dispatch, _: {}, { groupsService }: ServiceRepository) => {

        dispatch(startSubmit(CREATE_GROUP_FORM));

        try {

            const newGroup = await groupsService.create({ name });

            dispatch(dialogActions.CLOSE_DIALOG({ id: CREATE_GROUP_DIALOG }));
            dispatch(reset(CREATE_GROUP_FORM));
            dispatch(loadGroupsPanel());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `${newGroup.name} group has been created`,
                kind: SnackbarKind.SUCCESS
            }));

            return newGroup;

        } catch (e) {

            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(CREATE_GROUP_FORM, { name: 'Group with the same name already exists.' } as FormErrors));
            }

            return;

        }
    };
