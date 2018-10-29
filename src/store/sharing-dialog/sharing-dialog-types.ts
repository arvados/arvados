// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PermissionLevel } from '~/models/permission';

export const SHARING_DIALOG_NAME = 'SHARING_DIALOG_NAME';
export const SHARING_PUBLIC_ACCESS_FORM_NAME = 'SHARING_PUBLIC_ACCESS_FORM_NAME';
export const SHARING_MANAGEMENT_FORM_NAME = 'SHARING_MANAGEMENT_FORM_NAME';
export const SHARING_INVITATION_FORM_NAME = 'SHARING_INVITATION_FORM_NAME';

export interface SharingPublicAccessFormData {
    enabled: boolean;
    permissions: PermissionLevel;
}

export interface SharingManagementFormData {
    permissions: SharingManagementFormDataRow[];
}

export interface SharingManagementFormDataRow {
    email: string;
    permissions: PermissionLevel;
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