// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "../../models/kinds";
import { GroupContentsResource } from "../../services/groups-service/groups-service";

export interface ProjectPanelItem {
    uuid: string;
    name: string;
    kind: string;
    url: string;
    owner: string;
    lastModified: string;
    fileSize?: number;
    status?: string;
}


export function resourceToDataItem(r: GroupContentsResource): ProjectPanelItem {
    return {
        uuid: r.uuid,
        name: r.name,
        kind: r.kind,
        url: "",
        owner: r.ownerUuid,
        lastModified: r.modifiedAt,
        status:  r.kind === ResourceKind.Process ? r.state : undefined
    };
}

