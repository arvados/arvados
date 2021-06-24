// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from 'models/resource';

export enum KeyType {
    SSH = 'SSH'
}

export interface SshKeyResource extends Resource {
    name: string;
    keyType: KeyType;
    authorizedUserUuid: string;
    publicKey: string;
    expiresAt: string;
}