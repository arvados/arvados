// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupsService } from "~/services/groups-service/groups-service";
import { UserService } from '../user-service/user-service';
import { GroupResource } from '~/models/group';
import { UserResource } from '~/models/user';
import { extractUuidObjectType, ResourceObjectType, TrashableResource } from "~/models/resource";

export class AncestorService {
    constructor(
        private groupsService: GroupsService,
        private userService: UserService
    ) { }

    async ancestors(uuid: string, rootUuid: string): Promise<Array<UserResource | GroupResource | TrashableResource>> {
        const service = this.getService(extractUuidObjectType(uuid));
        if (service) {
            const resource = await service.get(uuid);
            if (uuid === rootUuid) {
                return [resource];
            } else {
                return [
                    ...await this.ancestors(resource.ownerUuid, rootUuid),
                    resource
                ];
            }
        } else {
            return [];
        }
    }

    private getService = (objectType?: string) => {
        switch (objectType) {
            case ResourceObjectType.GROUP:
                return this.groupsService;
            case ResourceObjectType.USER:
                return this.userService;
            default:
                return undefined;
        }
    }
}
