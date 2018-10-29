// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "~/services/link-service/link-service";
import { PermissionResource } from "~/models/permission";
import { ListArguments, ListResults } from '~/services/common-service/common-resource-service';
import { joinFilters, FilterBuilder } from '../api/filter-builder';
import { LinkClass } from '../../models/link';

export class PermissionService extends LinkService<PermissionResource> {

    list(args: ListArguments = {}): Promise<ListResults<PermissionResource>> {
        const { filters, ...other } = args;
        const classFilter = new FilterBuilder().addEqual('class', LinkClass.PERMISSION).getFilters();
        const newArgs = {
            ...other,
            filters: joinFilters(filters, classFilter),
        };
        return super.list(newArgs);
    }

}
