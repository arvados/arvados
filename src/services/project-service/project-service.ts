// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupsService } from "../groups-service/groups-service";
import { ProjectResource } from "~/models/project";
import { GroupClass } from "~/models/group";
import { ListArguments } from "~/services/common-service/common-service";
import { FilterBuilder, joinFilters } from "~/services/api/filter-builder";
import { TrashableResourceService } from '~/services/common-service/trashable-resource-service';
import { snakeCase } from 'lodash';
export class ProjectService extends GroupsService<ProjectResource> {

    create(data: Partial<ProjectResource>) {
        const projectData = { ...data, groupClass: GroupClass.PROJECT };
        return super.create(projectData);
    }

    list(args: ListArguments = {}) {
        return super.list({
            ...args,
            filters: joinFilters(
                args.filters,
                new FilterBuilder()
                    .addEqual("groupClass", GroupClass.PROJECT)
                    .getFilters()
            )
        });
    }
}
