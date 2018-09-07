// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from '../data-explorer/data-explorer-middleware-service';
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { Dispatch, MiddlewareAPI } from "redux";

export class SharedWithMeMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        return;
    }
}
