// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface ClientAuthorizationResource {
    uuid: string;
    apiToken: string;
    userId: number;
    createdByIpAddress: string;
    lastUsedByIpAddress: string;
    lastUsedAt: string;
    expiresAt: string;
    ownerUuid: string;
    scopes: string[];
}
