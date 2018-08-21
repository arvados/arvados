// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupContentsResource } from "~/services/groups-service/groups-service";
import { TrashResource } from "~/models/resource";

export interface TrashPanelItem {
    uuid: string;
    name: string;
    kind: string;
    owner: string;
    fileSize?: number;
    trashAt?: string;
    deleteAt?: string;
    isTrashed?: boolean;
}

export function resourceToDataItem(r: GroupContentsResource): TrashPanelItem {
    return {
        uuid: r.uuid,
        name: r.name,
        kind: r.kind,
        owner: r.ownerUuid,
        trashAt: (r as TrashResource).trashAt,
        deleteAt: (r as TrashResource).deleteAt,
        isTrashed: (r as TrashResource).isTrashed
    };
}
