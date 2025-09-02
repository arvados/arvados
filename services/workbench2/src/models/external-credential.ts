// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "./resource";

export interface ExternalCredential extends Resource {
    uuid: string;
    name: string;
    description: string;
    credentialClass: string;
    scopes?: string[];
    externalId: string;
    secret: string;
    expiresAt: string;
};

export const isExternalCredential = (obj: any): obj is ExternalCredential => {
    return obj
        && obj.uuid
        && obj.name
        && obj.description
        && obj.credentialClass
        && obj.externalId
        && obj.expiresAt;
};