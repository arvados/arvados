// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { Dispatch } from 'redux';
import { propertiesActions } from '~/store/properties/properties-actions';
import { getProperty } from '~/store/properties/properties';
import { Person } from '~/views-components/sharing-dialog/people-select';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { reset, startSubmit } from 'redux-form';
import { addGroupMember } from '~/store/groups-panel/groups-panel-actions';
import { getResource } from '~/store/resources/resources';
import { GroupResource } from '~/models/group';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';

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

export const addGroupMembers = ({ users }: AddGroupMembersFormData) =>

    async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {

        const groupUuid = getCurrentGroupDetailsPanelUuid(getState().properties);

        if (groupUuid) {

            dispatch(startSubmit(ADD_GROUP_MEMBERS_FORM));

            const group = getResource<GroupResource>(groupUuid)(getState().resources);

            for (const user of users) {

                await addGroupMember({
                    user,
                    group: {
                        uuid: groupUuid,
                        name: group ? group.name : groupUuid,
                    },
                    dispatch,
                    permissionService,
                });

            }

            dispatch(dialogActions.CLOSE_DIALOG({ id: ADD_GROUP_MEMBERS_FORM }));
            dispatch(GroupDetailsPanelActions.REQUEST_ITEMS());

        }
    };
