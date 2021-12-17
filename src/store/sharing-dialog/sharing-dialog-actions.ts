// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "store/dialog/dialog-actions";
import { withDialog } from "store/dialog/with-dialog";
import { SHARING_DIALOG_NAME, SharingPublicAccessFormData, SHARING_PUBLIC_ACCESS_FORM_NAME, SHARING_INVITATION_FORM_NAME, SharingManagementFormData, SharingInvitationFormData, VisibilityLevel, getSharingMangementFormData, getSharingPublicAccessFormData } from './sharing-dialog-types';
import { Dispatch } from 'redux';
import { ServiceRepository } from "services/services";
import { FilterBuilder } from 'services/api/filter-builder';
import { initialize, getFormValues, reset } from 'redux-form';
import { SHARING_MANAGEMENT_FORM_NAME } from 'store/sharing-dialog/sharing-dialog-types';
import { RootState } from 'store/store';
import { getDialog } from 'store/dialog/dialog-reducer';
import { PermissionLevel } from 'models/permission';
import { getPublicGroupUuid } from "store/workflow-panel/workflow-panel-actions";
import { PermissionResource } from 'models/permission';
import { differenceWith } from "lodash";
import { withProgress } from "store/progress-indicator/with-progress";
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { snackbarActions, SnackbarKind } from "../snackbar/snackbar-actions";
import { extractUuidKind, ResourceKind } from "models/resource";

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

interface SharingDialogData {
    resourceUuid: string;
    refresh: () => void;
}

const loadSharingDialog = async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {

    const dialog = getDialog<SharingDialogData>(getState().dialog, SHARING_DIALOG_NAME);
    if (dialog) {
        dispatch(progressIndicatorActions.START_WORKING(SHARING_DIALOG_NAME));
        try {
            const { items } = await permissionService.listResourcePermissions(dialog.data.resourceUuid);
            dispatch<any>(initializePublicAccessForm(items));
            await dispatch<any>(initializeManagementForm(items));
            dispatch(progressIndicatorActions.STOP_WORKING(SHARING_DIALOG_NAME));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'You do not have access to share this item', hideDuration: 2000, kind: SnackbarKind.ERROR }));
            dispatch(dialogActions.CLOSE_DIALOG({ id: SHARING_DIALOG_NAME }));
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


        if (visibility === VisibilityLevel.PRIVATE) {

            for (const permission of initialPermissions) {
                await permissionService.delete(permission.permissionUuid);
            }

        } else {

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
    }
};

const sendInvitations = async (_: Dispatch, getState: () => RootState, { permissionService, userService }: ServiceRepository) => {
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
