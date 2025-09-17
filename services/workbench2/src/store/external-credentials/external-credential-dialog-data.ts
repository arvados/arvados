// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface CreateExternalCredentialFormDialogData {
    name: string;
    description: string;
    credentialClass: string;
    externalId: string;
    expiresAt: string;
    secret: string;
    scopes?: string[];
}

export interface UpdateExternalCredentialFormDialogData {
    name: string;
    description: string;
    credentialClass: string;
    externalId: string;
    expiresAt: string;
    secret?: string;
    scopes?: string[];
}