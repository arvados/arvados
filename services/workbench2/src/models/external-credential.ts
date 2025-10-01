// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";

export interface ExternalCredential extends Resource {
    uuid: string;
    name: string;
    description?: string;
    credentialClass: string;
    scopes?: string[];
    externalId: string;
    secret: string;
    expiresAt: string;
    kind: ResourceKind.EXTERNAL_CREDENTIAL;
};

export const isExternalCredential = (res: Resource): res is ExternalCredential => {
    return res.kind === ResourceKind.EXTERNAL_CREDENTIAL;
};