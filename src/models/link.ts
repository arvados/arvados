import { Resource } from "./resource";

// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Link extends Resource {
    headUuid: string;
    tailUuid: string;
    linkClass: string;
    name: string;
    properties: {};
}
