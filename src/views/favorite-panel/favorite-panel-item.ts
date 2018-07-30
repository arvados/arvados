// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupContentsResource } from "../../services/groups-service/groups-service";
import { ResourceKind } from "../../models/resource";

export interface FavoritePanelItem {
    uuid: string;
    name: string;
    kind: string;
    url: string;
    owner: string;
    lastModified: string;
    fileSize?: number;
    status?: string;
}


export function resourceToDataItem(r: GroupContentsResource): FavoritePanelItem {
    return {
        uuid: r.uuid,
        name: r.name,
        kind: r.kind,
        url: "",
        owner: r.ownerUuid,
        lastModified: r.modifiedAt,
        status:  r.kind === ResourceKind.PROCESS ? r.state : undefined
    };
}

