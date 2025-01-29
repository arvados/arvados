// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProjectResource, isProjectResource } from "models/project";
import { Resource } from "models/resource";
import { getResource } from "store/resources/resources";
import { ResourcesState } from "store/resources/resources";
import { memoize } from "lodash";

export const resourceIsFrozen = memoize((resource: Resource, resources: ResourcesState): boolean => {
    let isFrozen: boolean = isProjectResource(resource) ? !!resource.frozenByUuid : false;
    let ownerUuid: string | undefined = resource?.ownerUuid;

    while(!isFrozen && !!ownerUuid && ownerUuid.indexOf('000000000000000') === -1) {
        const parentResource: ProjectResource | undefined = getResource<ProjectResource>(ownerUuid)(resources);
        isFrozen = !!parentResource?.frozenByUuid;
        ownerUuid = parentResource?.ownerUuid;
    }

    return isFrozen;
})