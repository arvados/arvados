// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";

export interface ExternalCredential extends Resource {
    kind: ResourceKind.EXTERNAL_CREDENTIAL;
    name: string;
    description?: string;
    credentialClass: string;
    scopes?: string[];
    externalId: string;
    secret: string;
    expiresAt: string;
};

export const isExternalCredential = (resource?: Resource): resource is ExternalCredential => {
    return !!resource && resource.kind === ResourceKind.EXTERNAL_CREDENTIAL;
};
