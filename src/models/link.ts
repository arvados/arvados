// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "./resource";
import { TagProperty } from "~/models/tag";

export interface LinkResource extends Resource {
    headUuid: string;
    tailUuid: string;
    linkClass: string;
    name: string;
    properties: TagProperty;
}

export enum LinkClass {
    STAR = 'star',
    TAG = 'tag',
    PERMISSION = 'permission',
}