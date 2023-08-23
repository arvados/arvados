// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind, ResourceWithProperties } from 'models/resource';

export interface LinkResource extends Resource, ResourceWithProperties {
    headUuid: string;
    headKind: ResourceKind;
    tailUuid: string;
    tailKind: string;
    linkClass: string;
    name: string;
    kind: ResourceKind.LINK;
}

export enum LinkClass {
    STAR = 'star',
    TAG = 'tag',
    PERMISSION = 'permission',
    PRESET = 'preset',
}
