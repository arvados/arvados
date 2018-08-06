// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "./resource";

export interface LinkResource extends Resource {
    headUuid: string;
    tailUuid: string;
    linkClass: string;
    name: string;
    properties: {
        key?: string;
        value?: any;
    };
}

export enum TailType {
    COLLECTION = 'Collection',
    JOB = 'Job'
}

export enum LinkClass {
    STAR = 'star',
    TAG = 'tag'
}