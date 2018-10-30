// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PermissionLevel } from '~/models/permission';
import { getFormValues, isDirty } from 'redux-form';
import { RootState } from '~/store/store';

export const SHARING_DIALOG_NAME = 'SHARING_DIALOG_NAME';
export const SHARING_PUBLIC_ACCESS_FORM_NAME = 'SHARING_PUBLIC_ACCESS_FORM_NAME';
export const SHARING_MANAGEMENT_FORM_NAME = 'SHARING_MANAGEMENT_FORM_NAME';
export const SHARING_INVITATION_FORM_NAME = 'SHARING_INVITATION_FORM_NAME';

export enum VisibilityLevel {
    PRIVATE = 'Private',
    SHARED = 'Shared',
    PUBLIC = 'Public',
}

export interface SharingPublicAccessFormData {
    visibility: VisibilityLevel;
    permissionUuid: string;
}

export interface SharingManagementFormData {
    permissions: SharingManagementFormDataRow[];
    initialPermissions: SharingManagementFormDataRow[];
}

export interface SharingManagementFormDataRow {
    email: string;
    permissions: PermissionLevel;
    permissionUuid: string;
}

export interface SharingInvitationFormData {
    permissions: PermissionLevel;
    invitedPeople: SharingInvitationFormPersonData[];
}

export interface SharingInvitationFormPersonData {
    email: string;
    name: string;
    uuid: string;
}

export const getSharingMangementFormData = (state: any) =>
    getFormValues(SHARING_MANAGEMENT_FORM_NAME)(state) as SharingManagementFormData;

export const getSharingPublicAccessFormData = (state: any) =>
    getFormValues(SHARING_PUBLIC_ACCESS_FORM_NAME)(state) as SharingPublicAccessFormData;

export const hasChanges = (state: RootState) =>
    isDirty(SHARING_PUBLIC_ACCESS_FORM_NAME)(state) ||
    isDirty(SHARING_MANAGEMENT_FORM_NAME)(state) ||
    isDirty(SHARING_INVITATION_FORM_NAME)(state);
