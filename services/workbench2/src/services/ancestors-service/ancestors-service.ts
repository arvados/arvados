// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupsService } from "services/groups-service/groups-service";
import { UserService } from '../user-service/user-service';
import { GroupResource } from 'models/group';
import { UserResource } from 'models/user';
import { extractUuidObjectType, ResourceObjectType } from "models/resource";
import { CollectionService } from "services/collection-service/collection-service";
import { CollectionResource } from "models/collection";

export class AncestorService {
    constructor(
        private groupsService: GroupsService,
        private userService: UserService,
        private collectionService: CollectionService,
    ) { }

    async ancestors(startUuid: string, endUuid: string): Promise<Array<UserResource | GroupResource | CollectionResource>> {
        return this._ancestors(startUuid, endUuid);
    }

    private async _ancestors(startUuid: string, endUuid: string, previousUuid = ''): Promise<Array<UserResource | GroupResource | CollectionResource>> {

        if (startUuid === previousUuid) {
            return [];
        }

        const service = this.getService(extractUuidObjectType(startUuid));
        if (service) {
            try {
                const resource = await service.get(startUuid, false);
                if (startUuid === endUuid) {
                    return [resource];
                } else {
                    return [
                        ...await this._ancestors(resource.ownerUuid, endUuid, startUuid),
                        resource
                    ];
                }
            } catch (e) {
                return [];
            }
        }
        return [];
    }

    private getService = (objectType?: string) => {
        switch (objectType) {
            case ResourceObjectType.GROUP:
                return this.groupsService;
            case ResourceObjectType.USER:
                return this.userService;
            case ResourceObjectType.COLLECTION:
                return this.collectionService;
            default:
                return undefined;
        }
    }
}
