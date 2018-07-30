// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getDataExplorer } from "./data-explorer-reducer";
import { MiddlewareAPI } from "redux";
import { DataColumns } from "../../components/data-table/data-table";

export abstract class DataExplorerMiddlewareService {
    protected readonly id: string;

    protected constructor(id: string) {
        this.id = id;
    }

    public getId() {
        return this.id;
    }

    abstract getColumns(): DataColumns<any>;
    abstract requestItems(api: MiddlewareAPI): void;

    getDataExplorer(api: MiddlewareAPI) {
        return getDataExplorer(api.getState().dataExplorer, this.id);
    }
}
