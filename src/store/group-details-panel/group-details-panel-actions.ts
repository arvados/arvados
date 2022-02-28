// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { Dispatch } from 'redux';
import { propertiesActions } from 'store/properties/properties-actions';
import { getProperty } from 'store/properties/properties';
import { dialogActions } from 'store/dialog/dialog-actions';
import { deleteGroupMember } from 'store/groups-panel/groups-panel-actions';
import { getResource } from 'store/resources/resources';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { PermissionResource, PermissionLevel } from 'models/permission';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { LinkResource } from 'models/link';
import { deleteResources, updateResources } from 'store/resources/resources-actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';
// import { UserProfileGroupsActions } from 'store/user-profile/user-profile-actions';

export const GROUP_DETAILS_MEMBERS_PANEL_ID = 'groupDetailsMembersPanel';
export const GROUP_DETAILS_PERMISSIONS_PANEL_ID = 'groupDetailsPermissionsPanel';
export const MEMBER_ATTRIBUTES_DIALOG = 'memberAttributesDialog';
export const MEMBER_REMOVE_DIALOG = 'memberRemoveDialog';

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

export const openAddGroupMembersDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const groupUuid = getCurrentGroupDetailsPanelUuid(getState().properties);
        if (groupUuid) {
            dispatch<any>(openSharingDialog(groupUuid, () => {
                dispatch(GroupMembersPanelActions.REQUEST_ITEMS());
            }));
        }
    };

export const editPermissionLevel = (uuid: string, level: PermissionLevel) =>
    async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
        try {
            const permission = await permissionService.update(uuid, {name: level});
            dispatch(updateResources([permission]));
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
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await deleteGroupMember({
            link: {
                uuid,
            },
            permissionService,
            dispatch,
        });
        dispatch<any>(deleteResources([uuid]));
        dispatch(GroupMembersPanelActions.REQUEST_ITEMS());
        // dispatch(UserProfileGroupsActions.REQUEST_ITEMS());

        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    };

export const setMemberIsHidden = (memberLinkUuid: string, permissionLinkUuid: string, visible: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
        const memberLink = getResource<LinkResource>(memberLinkUuid)(getState().resources);

        if (!visible && permissionLinkUuid) {
            // Remove read permission
            try {
                await permissionService.delete(permissionLinkUuid);
                dispatch<any>(deleteResources([permissionLinkUuid]));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Removed read permission.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS,
                }));
            } catch (e) {
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Failed to remove permission',
                    kind: SnackbarKind.ERROR,
                }));
            }
        } else if (visible && memberLink) {
            // Create read permission
            try {
                const permission = await permissionService.create({
                    headUuid: memberLink.tailUuid,
                    tailUuid: memberLink.headUuid,
                    name: PermissionLevel.CAN_READ,
                });
                dispatch(updateResources([permission]));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Created read permission.',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS,
                }));
            } catch(e) {
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Failed to create permission',
                    kind: SnackbarKind.ERROR,
                }));
            }
        }
    };
