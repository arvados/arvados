// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { Dispatch } from 'redux';
import { propertiesActions } from '~/store/properties/properties-actions';
import { getProperty } from '~/store/properties/properties';
import { Person } from '~/views-components/sharing-dialog/people-select';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { reset } from 'redux-form';

export const GROUP_DETAILS_PANEL_ID = 'groupDetailsPanel';
export const ADD_GROUP_MEMBERS_DIALOG = 'addGrupMembers';
export const ADD_GROUP_MEMBERS_FORM = 'addGrupMembers';
export const ADD_GROUP_MEMBERS_USERS_FIELD_NAME = 'users';


export const GroupDetailsPanelActions = bindDataExplorerActions(GROUP_DETAILS_PANEL_ID);

export const loadGroupDetailsPanel = (groupUuid: string) =>
    (dispatch: Dispatch) => {
        dispatch(propertiesActions.SET_PROPERTY({ key: GROUP_DETAILS_PANEL_ID, value: groupUuid }));
        dispatch(GroupDetailsPanelActions.REQUEST_ITEMS());
    };

export const getCurrentGroupDetailsPanelUuid = getProperty<string>(GROUP_DETAILS_PANEL_ID);

export interface AddGroupMembersFormData {
    [ADD_GROUP_MEMBERS_USERS_FIELD_NAME]: Person[];
}

export const openAddGroupMembersDialog = () =>
    (dispatch: Dispatch) => {
        dispatch(dialogActions.OPEN_DIALOG({ id: ADD_GROUP_MEMBERS_DIALOG, data: {} }));
        dispatch(reset(ADD_GROUP_MEMBERS_FORM));
    };
