// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupsService, ContentsArguments } from "../groups-service/groups-service";
import { ProjectResource } from "../../models/project";
import { GroupClass } from "../../models/group";
import { ListArguments } from "../../common/api/common-resource-service";
import { FilterBuilder } from "../../common/api/filter-builder";

export class ProjectService extends GroupsService<ProjectResource> {

    create(data: Partial<ProjectResource>) {
        const projectData = { ...data, groupClass: GroupClass.Project };
        return super.create(projectData);
    }

    list(args: ListArguments = {}) {
        return super.list({
            ...args,
            filters: this.addProjectFilter(args.filters)
        });
    }

    private addProjectFilter(filters?: FilterBuilder) {
        return FilterBuilder
            .create()
            .concat(filters
                ? filters
                : FilterBuilder.create())
            .concat(FilterBuilder
                .create<ProjectResource>()
                .addEqual("groupClass", GroupClass.Project));
    }

}
