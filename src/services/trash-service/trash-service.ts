// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupsService } from "../groups-service/groups-service";
import { AxiosInstance } from "axios";

export class TrashService extends GroupsService {
    constructor(serverApi: AxiosInstance) {
        super(serverApi);
    }
}
