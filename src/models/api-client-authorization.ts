// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface ApiClientAuthorization {
    uuid: string;
    apiToken: string;
    apiClientId: number;
    userId: number;
    createdByIpAddress: string;
    lastUsedByIpAddress: string;
    lastUsedAt: string;
    expiresAt: string;
    createdAt: string;
    updatedAt: string;
    ownerUuid: string;
    defaultOwnerUuid: string;
    scopes: string[];
}