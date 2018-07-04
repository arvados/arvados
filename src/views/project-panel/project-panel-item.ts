// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "../../common/api/common-resource-service";
import { DataItem } from "../../components/data-table/data-table";

export interface ProjectPanelItem extends DataItem {
    uuid: string;
    name: string;
    kind: string;
    url: string;
    owner: string;
    lastModified: string;
    fileSize?: number;
    status?: string;
}

export function resourceToDataItem(r: Resource): ProjectPanelItem {
    return {
        key: r.uuid,
        uuid: r.uuid,
        name: r.uuid,
        kind: r.kind,
        url: "",
        owner: r.ownerUuid,
        lastModified: r.modifiedAt
    };
}

