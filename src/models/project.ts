// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Project {
    name: string;
    createdAt: string;
    modifiedAt: string;
    uuid: string;
    ownerUuid: string;
    href: string;
}
