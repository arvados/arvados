// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "store/dialog/dialog-actions";
import { withDialog } from "store/dialog/with-dialog";
import {
    SHARING_DIALOG_NAME,
    SHARING_INVITATION_FORM_NAME,
    SharingManagementFormData,
    SharingInvitationFormData,
    getSharingMangementFormData,
} from './sharing-dialog-types';
import { Dispatch } from 'redux';
import { ServiceRepository } from "services/services";
import { FilterBuilder } from 'services/api/filter-builder';
import { initialize, getFormValues, reset } from 'redux-form';
import { SHARING_MANAGEMENT_FORM_NAME } from 'store/sharing-dialog/sharing-dialog-types';
import { RootState } from 'store/store';
import { getDialog } from 'store/dialog/dialog-reducer';
import { PermissionLevel } from 'models/permission';
import { PermissionResource } from 'models/permission';
import { differenceWith } from "lodash";
import { withProgress } from "store/progress-indicator/with-progress";
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { snackbarActions, SnackbarKind } from "../snackbar/snackbar-actions";
import {
    extractUuidKind,
    extractUuidObjectType,
    ResourceKind,
    ResourceObjectType
} from "models/resource";
import { resourcesActions } from "store/resources/resources-actions";

export const openSharingDialog = (resourceUuid: string, refresh?: () => void) =>
    (dispatch: Dispatch) => {
        dispatch(dialogActions.OPEN_DIALOG({ id: SHARING_DIALOG_NAME, data: {resourceUuid, refresh} }));
        dispatch<any>(loadSharingDialog);
    };

export const closeSharingDialog = () =>
    dialogActions.CLOSE_DIALOG({ id: SHARING_DIALOG_NAME });

export const connectSharingDialog = withDialog(SHARING_DIALOG_NAME);
export const connectSharingDialogProgress = withProgress(SHARING_DIALOG_NAME);


export const saveSharingDialogChanges = async (dispatch: Dispatch, getState: () => RootState) => {
    dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
    await dispatch<any>(saveManagementChanges);
    await dispatch<any>(sendInvitations);
    dispatch(reset(SHARING_INVITATION_FORM_NAME));
    await dispatch<any>(loadSharingDialog);
    dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));

    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog && dialog.data.refresh) {
        dialog.data.refresh();
    }
};

export const sendSharingInvitations = async (dispatch: Dispatch, getState: () => RootState) => {
    dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
    await dispatch<any>(sendInvitations);
    dispatch(closeSharingDialog());
    dispatch(snackbarActions.OPEN_SNACKBAR({
        message: 'Resource has been shared',
        kind: SnackbarKind.SUCCESS,
    }));
    dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));

    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog && dialog.data.refresh) {
        dialog.data.refresh();
    }
};

export interface SharingDialogData {
    resourceUuid: string;
    refresh: () => void;
}

export const createSharingToken = async (dispatch: Dispatch, getState: () => RootState, { apiClientAuthorizationService }: ServiceRepository) => {
    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog) {
        const resourceUuid = dialog.data.resourceUuid;
        if (extractUuidObjectType(resourceUuid) === ResourceObjectType.COLLECTION) {
            dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
            try {
                const sharingToken = await apiClientAuthorizationService.createCollectionSharingToken(resourceUuid);
                dispatch(resourcesActions.SET_RESOURCES([sharingToken]));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Sharing URL created',
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS,
                }));
            } catch (e) {
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Failed to create sharing URL',
                    hideDuration: 2000,
                    kind: SnackbarKind.ERROR,
                }));
            } finally {
                dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));
            }
        }
    }
};

export const deleteSharingToken = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, { apiClientAuthorizationService }: ServiceRepository) => {
    dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
    try {
        await apiClientAuthorizationService.delete(uuid);
        dispatch(resourcesActions.DELETE_RESOURCES([uuid]));
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: 'Sharing URL removed',
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS,
        }));
    } catch (e) {
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: 'Failed to remove sharing URL',
            hideDuration: 2000,
            kind: SnackbarKind.ERROR,
        }));
    } finally {
        dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));
    }
};

const loadSharingDialog = async (dispatch: Dispatch, getState: () => RootState, { permissionService, apiClientAuthorizationService }: ServiceRepository) => {

    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog) {
        dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
        try {
            const resourceUuid = dialog.data.resourceUuid;
            const { items } = await permissionService.listResourcePermissions(resourceUuid);
            await dispatch<any>(initializeManagementForm(items));
            // For collections, we need to load the public sharing tokens
            if (extractUuidObjectType(resourceUuid) === ResourceObjectType.COLLECTION) {
                const sharingTokens = await apiClientAuthorizationService.listCollectionSharingTokens(resourceUuid);
                dispatch(resourcesActions.SET_RESOURCES([...sharingTokens.items]));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'You do not have access to share this item',
                hideDuration: 2000,
                kind: SnackbarKind.ERROR }));
            dispatch(dialogActions.CLOSE_DIALOG({ id: SHARING_DIALOG_NAME }));
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));
        }
    }
};

const initializeManagementForm = (permissionLinks: PermissionResource[]) =>
    async (dispatch: Dispatch, getState: () => RootState, { userService, groupsService }: ServiceRepository) => {

        const filters = new FilterBuilder()
            .addIn('uuid', permissionLinks.map(({ tailUuid }) => tailUuid))
            .getFilters();

        const { items: users } = await userService.list({ filters, count: "none" });
        const { items: groups } = await groupsService.list({ filters, count: "none" });

        const getEmail = (tailUuid: string) => {
            const user = users.find(({ uuid }) => uuid === tailUuid);
            const group = groups.find(({ uuid }) => uuid === tailUuid);
            return user
                ? user.email
                : group
                    ? group.name
                    : tailUuid;
        };

        const managementPermissions = permissionLinks
            .map(({ tailUuid, name, uuid }) => ({
                email: getEmail(tailUuid),
                permissions: name as PermissionLevel,
                permissionUuid: uuid,
            }));

        const managementFormData: SharingManagementFormData = {
            permissions: managementPermissions,
            initialPermissions: managementPermissions,
        };

        dispatch(initialize(SHARING_MANAGEMENT_FORM_NAME, managementFormData));
    };

const saveManagementChanges = async (_: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<string>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {
        const { initialPermissions, permissions } = getSharingMangementFormData(state);
        const cancelledPermissions = differenceWith(
            initialPermissions,
            permissions,
            (a, b) => a.permissionUuid === b.permissionUuid
        );

        for (const { permissionUuid } of cancelledPermissions) {
            await permissionService.delete(permissionUuid);
        }

        for (const permission of permissions) {
            await permissionService.update(permission.permissionUuid, { name: permission.permissions });
        }
    }
};

const sendInvitations = async (_: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<SharingDialogData>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {
        const invitations = getFormValues(SHARING_INVITATION_FORM_NAME)(state) as SharingInvitationFormData;

        const getGroupsFromForm = invitations.invitedPeople.filter((invitation) => extractUuidKind(invitation.uuid) === ResourceKind.GROUP);
        const getUsersFromForm = invitations.invitedPeople.filter((invitation) => extractUuidKind(invitation.uuid) === ResourceKind.USER);

        const invitationDataUsers = getUsersFromForm
            .map(person => ({
                ownerUuid: user.uuid,
                headUuid: dialog.data.resourceUuid,
                tailUuid: person.uuid,
                name: invitations.permissions
            }));

        const invitationsDataGroups = getGroupsFromForm.map(
            group => ({
                ownerUuid: user.uuid,
                headUuid: dialog.data.resourceUuid,
                tailUuid: group.uuid,
                name: invitations.permissions
            })
        );

        const data = invitationDataUsers.concat(invitationsDataGroups);

        for (const invitation of data) {
            await permissionService.create(invitation);
        }
    }
};
