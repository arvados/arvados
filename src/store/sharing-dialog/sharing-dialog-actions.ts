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
    SharingPublicAccessFormData,
    VisibilityLevel,
    SHARING_PUBLIC_ACCESS_FORM_NAME,
} from './sharing-dialog-types';
import { Dispatch } from 'redux';
import { ServiceRepository } from "services/services";
import { FilterBuilder } from 'services/api/filter-builder';
import { initialize, getFormValues, reset } from 'redux-form';
import { SHARING_MANAGEMENT_FORM_NAME } from 'store/sharing-dialog/sharing-dialog-types';
import { RootState } from 'store/store';
import { getDialog } from 'store/dialog/dialog-reducer';
import { PermissionLevel, PermissionResource } from 'models/permission';
import { differenceWith } from "lodash";
import { withProgress } from "store/progress-indicator/with-progress";
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { snackbarActions, SnackbarKind } from "../snackbar/snackbar-actions";
import {
    extractUuidObjectType,
    ResourceObjectType
} from "models/resource";
import { resourcesActions } from "store/resources/resources-actions";
import { getPublicGroupUuid } from "store/workflow-panel/workflow-panel-actions";
import { getSharingPublicAccessFormData } from './sharing-dialog-types';

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
    await dispatch<any>(savePublicPermissionChanges);
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

export interface SharingDialogData {
    resourceUuid: string;
    refresh: () => void;
}

export const createSharingToken = (expDate: Date | undefined) => async (dispatch: Dispatch, getState: () => RootState, { apiClientAuthorizationService }: ServiceRepository) => {
    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog) {
        const resourceUuid = dialog.data.resourceUuid;
        if (extractUuidObjectType(resourceUuid) === ResourceObjectType.COLLECTION) {
            dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
            try {
                const sharingToken = await apiClientAuthorizationService.createCollectionSharingToken(resourceUuid, expDate);
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

const loadSharingDialog = async (dispatch: Dispatch, getState: () => RootState, { apiClientAuthorizationService }: ServiceRepository) => {

    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog) {
        dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
        try {
            const resourceUuid = dialog.data.resourceUuid;
            await dispatch<any>(initializeManagementForm);
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

export const initializeManagementForm = async (dispatch: Dispatch, getState: () => RootState, { userService, groupsService, permissionService }: ServiceRepository) => {

        const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
        if (!dialog) {
            return;
        }
        dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
        const resourceUuid = dialog?.data.resourceUuid;
        const { items: permissionLinks } = await permissionService.listResourcePermissions(resourceUuid);
        dispatch<any>(initializePublicAccessForm(permissionLinks));
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
            .filter(item =>
                item.tailUuid !== getPublicGroupUuid(getState()))
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
        dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));
    };

const initializePublicAccessForm = (permissionLinks: PermissionResource[]) =>
    (dispatch: Dispatch, getState: () => RootState, ) => {
        const [publicPermission] = permissionLinks
            .filter(item => item.tailUuid === getPublicGroupUuid(getState()));
        const publicAccessFormData: SharingPublicAccessFormData = publicPermission
            ? {
                visibility: VisibilityLevel.PUBLIC,
                permissionUuid: publicPermission.uuid,
            }
            : {
                visibility: permissionLinks.length > 0
                    ? VisibilityLevel.SHARED
                    : VisibilityLevel.PRIVATE,
                permissionUuid: '',
            };
        dispatch(initialize(SHARING_PUBLIC_ACCESS_FORM_NAME, publicAccessFormData));
    };

const savePublicPermissionChanges = async (_: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<SharingDialogData>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {
        const { permissionUuid, visibility } = getSharingPublicAccessFormData(state);
        if (permissionUuid) {
            if (visibility === VisibilityLevel.PUBLIC) {
                await permissionService.update(permissionUuid, {
                    name: PermissionLevel.CAN_READ
                });
            } else {
                await permissionService.delete(permissionUuid);
            }
        } else if (visibility === VisibilityLevel.PUBLIC) {
            await permissionService.create({
                ownerUuid: user.uuid,
                headUuid: dialog.data.resourceUuid,
                tailUuid: getPublicGroupUuid(state),
                name: PermissionLevel.CAN_READ,
            });
        }
    }
};

const saveManagementChanges = async (_: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<string>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {
        const { initialPermissions, permissions } = getSharingMangementFormData(state);
        const { visibility } = getSharingPublicAccessFormData(state);
        const cancelledPermissions = visibility === VisibilityLevel.PRIVATE
            ? initialPermissions
            : differenceWith(
                initialPermissions,
                permissions,
                (a, b) => a.permissionUuid === b.permissionUuid
            );

        const deletions = cancelledPermissions.map(({ permissionUuid }) =>
            permissionService.delete(permissionUuid));
        const updates = permissions.map(update =>
            permissionService.update(update.permissionUuid, { name: update.permissions }));
        await Promise.all([...deletions, ...updates]);
    }
};

const sendInvitations = async (_: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<SharingDialogData>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {
        const invitations = getFormValues(SHARING_INVITATION_FORM_NAME)(state) as SharingInvitationFormData;
        const data = invitations.invitedPeople.map(invitee => ({
            ownerUuid: user.uuid,
            headUuid: dialog.data.resourceUuid,
            tailUuid: invitee.uuid,
            name: invitations.permissions
        }));
        const changes = data.map( invitation => permissionService.create(invitation));
        await Promise.all(changes);
    }
};
