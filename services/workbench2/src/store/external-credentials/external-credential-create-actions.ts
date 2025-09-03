// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface ExternalCredentialCreateFormDialogData {
    name: string;
    description: string;
    credentialClass: string;
    externalId: string;
    expiresAt: string;
    scopes: string[];
}