// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupsService } from "../groups-service/groups-service";
import { TrashService } from "./trash-service";
import { mockResourceService } from "~/common/api/common-resource-service.test";

describe("TrashService", () => {

    let groupService: GroupsService;

    beforeEach(() => {
        groupService = mockResourceService(GroupsService);
    });

});
