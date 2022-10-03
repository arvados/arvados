// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProjectResource } from "models/project";
import { getResource } from "store/resources/resources";

export const resourceIsFrozen = (resource: any, resources): boolean => {
    let isFrozen: boolean = !!resource.frozenByUuid;
    let ownerUuid: string | undefined = resource?.ownerUuid;

    while(!isFrozen && !!ownerUuid && resource.uuid !== ownerUuid) {
        const parentResource: ProjectResource | undefined = getResource<ProjectResource>(ownerUuid)(resources);
        isFrozen = !!parentResource?.frozenByUuid;
        ownerUuid = parentResource?.ownerUuid;
    }

    return isFrozen;
}