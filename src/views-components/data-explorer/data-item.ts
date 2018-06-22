// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getResourceKind, Resource, ResourceKind } from "../../models/resource";

export interface DataItem {
    uuid: string;
    name: string;
    type: ResourceKind;
    url: string;
    owner: string;
    lastModified: string;
    fileSize?: number;
    status?: string;
}

function resourceToDataItem(r: Resource, kind?: ResourceKind) {
    return {
        uuid: r.uuid,
        name: r.name,
        type: kind ? kind : getResourceKind(r.kind),
        owner: r.ownerUuid,
        lastModified: r.modifiedAt
    };
}

