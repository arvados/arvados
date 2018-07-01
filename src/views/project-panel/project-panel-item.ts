// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { getResourceKind, Resource, ResourceKind } from "../../models/resource";

export interface ProjectPanelItem {
    uuid: string;
    name: string;
    kind: ResourceKind;
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
        kind: kind ? kind : getResourceKind(r.kind),
        owner: r.ownerUuid,
        lastModified: r.modifiedAt
    };
}

