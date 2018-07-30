// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getDataExplorer } from "./data-explorer-reducer";
import { MiddlewareAPI } from "redux";
import { DataColumns } from "../../components/data-table/data-table";

export abstract class DataExplorerMiddlewareService {
    protected api: MiddlewareAPI;
    protected readonly id: string;

    protected constructor(id: string) {
        this.id = id;
    }

    public getId() {
        return this.id;
    }

    abstract getColumns(): DataColumns<any>;
    abstract requestItems(api: MiddlewareAPI): void;

    setApi(api: MiddlewareAPI) {
        this.api = api;
    }
    getDataExplorer() {
        return getDataExplorer(this.api.getState().dataExplorer, this.id);
    }
}
