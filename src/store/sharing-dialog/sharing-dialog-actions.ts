// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "~/store/dialog/dialog-actions";
import { withDialog } from "~/store/dialog/with-dialog";
import { SHARING_DIALOG_NAME, SharingPublicAccessFormData, SHARING_PUBLIC_ACCESS_FORM_NAME, SHARING_INVITATION_FORM_NAME, SharingManagementFormData, SharingInvitationFormData } from './sharing-dialog-types';
import { Dispatch } from 'redux';
import { ServiceRepository } from "~/services/services";
import { FilterBuilder } from '~/services/api/filter-builder';
import { initialize, getFormValues, isDirty, reset } from 'redux-form';
import { SHARING_MANAGEMENT_FORM_NAME } from '~/store/sharing-dialog/sharing-dialog-types';
import { RootState } from '~/store/store';
import { getDialog } from '~/store/dialog/dialog-reducer';
import { PermissionLevel } from '../../models/permission';
import { getPublicGroupUuid } from "~/store/workflow-panel/workflow-panel-actions";

export const openSharingDialog = (resourceUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {

        dispatch(dialogActions.OPEN_DIALOG({ id: SHARING_DIALOG_NAME, data: resourceUuid }));

        const state = getState();
        const { items } = await permissionService.listResourcePermissions(resourceUuid);

        const managementFormData: SharingManagementFormData = {
            permissions: items
                .filter(item =>
                    item.tailUuid !== getPublicGroupUuid(state))
                .map(({ tailUuid, name }) => ({
                    email: tailUuid,
                    permissions: name as PermissionLevel,
                }))
        };

        dispatch(initialize(SHARING_MANAGEMENT_FORM_NAME, managementFormData));

        const [publicPermission] = items.filter(item => item.tailUuid === getPublicGroupUuid(state));
        if (publicPermission) {
            const publicAccessFormData: SharingPublicAccessFormData = {
                enabled: publicPermission.name !== PermissionLevel.NONE,
                permissions: publicPermission.name as PermissionLevel,
            };

            dispatch(initialize(SHARING_PUBLIC_ACCESS_FORM_NAME, publicAccessFormData));
        } else {
            dispatch(reset(SHARING_PUBLIC_ACCESS_FORM_NAME));
        }
    };

export const closeSharingDialog = () =>
    dialogActions.CLOSE_DIALOG({ id: SHARING_DIALOG_NAME });

export const connectSharingDialog = withDialog(SHARING_DIALOG_NAME);

export const saveSharingDialogChanges = async (dispatch: Dispatch) => {
    dispatch<any>(savePublicPermissionChanges);
    dispatch<any>(sendInvitations);
};

const savePublicPermissionChanges = async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<string>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {
        const publicAccess = getFormValues(SHARING_PUBLIC_ACCESS_FORM_NAME)(state) as SharingPublicAccessFormData;

        const filters = new FilterBuilder()
            .addEqual('headUuid', dialog.data)
            .getFilters();

        const { items } = await permissionService.list({ filters });

        const [publicPermission] = items.filter(item => item.tailUuid === getPublicGroupUuid(state));

        if (publicPermission) {

            await permissionService.update(publicPermission.uuid, {
                name: publicAccess.enabled ? publicAccess.permissions : PermissionLevel.NONE
            });

        } else {

            await permissionService.create({
                ownerUuid: user.uuid,
                headUuid: dialog.data,
                tailUuid: getPublicGroupUuid(state),
                name: publicAccess.enabled ? publicAccess.permissions : PermissionLevel.NONE
            });
        }
    }
};

const sendInvitations = async (dispatch: Dispatch, getState: () => RootState, { permissionService }: ServiceRepository) => {
    const state = getState();
    const { user } = state.auth;
    const dialog = getDialog<string>(state.dialog, SHARING_DIALOG_NAME);
    if (dialog && user) {

        const invitations = getFormValues(SHARING_INVITATION_FORM_NAME)(state) as SharingInvitationFormData;

        const promises = invitations.invitedPeople
            .map(person => ({
                ownerUuid: user.uuid,
                headUuid: dialog.data,
                tailUuid: person.uuid,
                name: invitations.permissions
            }))
            .map(data => permissionService.create(data));

        await Promise.all(promises);
    }
};

export const hasChanges = (state: RootState) =>
    isDirty(SHARING_PUBLIC_ACCESS_FORM_NAME)(state) ||
    isDirty(SHARING_MANAGEMENT_FORM_NAME)(state) ||
    isDirty(SHARING_INVITATION_FORM_NAME)(state);
