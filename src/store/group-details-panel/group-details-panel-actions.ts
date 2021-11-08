// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { Dispatch } from 'redux';
import { propertiesActions } from 'store/properties/properties-actions';
import { getProperty } from 'store/properties/properties';
import { Participant } from 'views-components/sharing-dialog/participant-select';
import { dialogActions } from 'store/dialog/dialog-actions';
import { initialize, reset, startSubmit } from 'redux-form';
import { addGroupMember, deleteGroupMember } from 'store/groups-panel/groups-panel-actions';
import { getResource } from 'store/resources/resources';
import { GroupResource } from 'models/group';
import { Resource } from 'models/resource';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { PermissionResource, PermissionLevel } from 'models/permission';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { PermissionSelectValue, parsePermissionLevel, formatPermissionLevel } from 'views-components/sharing-dialog/permission-select';

export const GROUP_DETAILS_MEMBERS_PANEL_ID = 'groupDetailsMembersPanel';
export const GROUP_DETAILS_PERMISSIONS_PANEL_ID = 'groupDetailsPermissionsPanel';
export const ADD_GROUP_MEMBERS_DIALOG = 'addGroupMembers';
export const ADD_GROUP_MEMBERS_FORM = 'addGroupMembers';
export const ADD_GROUP_MEMBERS_USERS_FIELD_NAME = 'users';
export const MEMBER_ATTRIBUTES_DIALOG = 'memberAttributesDialog';
export const MEMBER_REMOVE_DIALOG = 'memberRemoveDialog';
export const EDIT_PERMISSION_LEVEL_DIALOG = 'editPermissionLevel';
export const EDIT_PERMISSION_LEVEL_FORM = 'editPermissionLevel';
export const EDIT_PERMISSION_LEVEL_FIELD_NAME = 'name';
export const EDIT_PERMISSION_LEVEL_UUID_FIELD_NAME = 'uuid';

export const GroupMembersPanelActions = bindDataExplorerActions(GROUP_DETAILS_MEMBERS_PANEL_ID);
export const GroupPermissionsPanelActions = bindDataExplorerActions(GROUP_DETAILS_PERMISSIONS_PANEL_ID);

export const loadGroupDetailsPanel = (groupUuid: string) =>
    (dispatch: Dispatch) => {
        dispatch(propertiesActions.SET_PROPERTY({ key: GROUP_DETAILS_MEMBERS_PANEL_ID, value: groupUuid }));
        dispatch(GroupMembersPanelActions.REQUEST_ITEMS());
        dispatch(propertiesActions.SET_PROPERTY({ key: GROUP_DETAILS_PERMISSIONS_PANEL_ID, value: groupUuid }));
        dispatch(GroupPermissionsPanelActions.REQUEST_ITEMS());
    };

export const getCurrentGroupDetailsPanelUuid = getProperty<string>(GROUP_DETAILS_MEMBERS_PANEL_ID);

export interface AddGroupMembersFormData {
    [ADD_GROUP_MEMBERS_USERS_FIELD_NAME]: Participant[];
}

export interface EditPermissionLevelFormData {
    [EDIT_PERMISSION_LEVEL_UUID_FIELD_NAME]: string;
    [EDIT_PERMISSION_LEVEL_FIELD_NAME]: PermissionSelectValue;
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
            dispatch(GroupMembersPanelActions.REQUEST_ITEMS());

        }
    };

export const openEditPermissionLevelDialog = (linkUuid: string, resourceUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const link = getResource<PermissionResource>(linkUuid)(getState().resources);
        const resource = getResource<Resource>(resourceUuid)(getState().resources);

        if (link) {
            dispatch(reset(EDIT_PERMISSION_LEVEL_FORM));
            dispatch<any>(initialize(EDIT_PERMISSION_LEVEL_FORM, {[EDIT_PERMISSION_LEVEL_UUID_FIELD_NAME]: link.uuid, [EDIT_PERMISSION_LEVEL_FIELD_NAME]: formatPermissionLevel(link.name as PermissionLevel)}));
            dispatch(dialogActions.OPEN_DIALOG({ id: EDIT_PERMISSION_LEVEL_DIALOG, data: resource }));
        }
    };

export const editPermissionLevel = (data: EditPermissionLevelFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
        try {
            await permissionService.update(data[EDIT_PERMISSION_LEVEL_UUID_FIELD_NAME], {name: parsePermissionLevel(data[EDIT_PERMISSION_LEVEL_FIELD_NAME])});
            dispatch(dialogActions.CLOSE_DIALOG({ id: EDIT_PERMISSION_LEVEL_DIALOG }));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Permission level changed.', hideDuration: 2000 }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'Failed to update permission',
                kind: SnackbarKind.ERROR,
            }));
        }
    };

export const openGroupMemberAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<PermissionResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: MEMBER_ATTRIBUTES_DIALOG, data }));
    };

export const openRemoveGroupMemberDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: MEMBER_REMOVE_DIALOG,
            data: {
                title: 'Remove member',
                text: 'Are you sure you want to remove this member from this group?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeGroupMember = (uuid: string) =>

    async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {

        const groupUuid = getCurrentGroupDetailsPanelUuid(getState().properties);

        if (groupUuid) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));

            await deleteGroupMember({
                link: {
                    uuid,
                },
                permissionService,
                dispatch,
            });

            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            dispatch(GroupMembersPanelActions.REQUEST_ITEMS());

        }

    };
