// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "~/services/link-service/link-service";
import { PermissionResource } from "~/models/permission";
import { CommonResourceService } from '~/services/common-service/common-resource-service';
import { LinkClass } from '../../models/link';
import { ListArguments, ListResults } from '~/services/common-service/common-service';

export class PermissionService extends LinkService<PermissionResource> {

    permissionListService = new CommonResourceService(this.serverApi, 'permissions', this.actions);
    create(data?: Partial<PermissionResource>) {
        return super.create({ ...data, linkClass: LinkClass.PERMISSION });
    }

    listResourcePermissions(uuid: string, args: ListArguments = {}): Promise<ListResults<PermissionResource>> {
        const service = new CommonResourceService<PermissionResource>(this.serverApi, `permissions/${uuid}`, this.actions);
        return service.list(args);
    }

}
