// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource } from "./link";

export interface TagResource extends LinkResource {
    tailUuid: TagTailType;
    properties: TagProperty;
}

export interface TagProperty {
    key: string;
    value: string;
}

export enum TagTailType {
    COLLECTION = 'Collection',
    JOB = 'Job'
}