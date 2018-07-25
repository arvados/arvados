// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getDataExplorer } from "./data-explorer-reducer";
import { MiddlewareAPI } from "../../../node_modules/redux";
import { DataColumns } from "../../components/data-table/data-table";

export abstract class DataExplorerMiddlewareService {

    abstract get Id(): string;
    abstract get Columns(): DataColumns<any>;
    abstract requestItems (api: MiddlewareAPI): void;
    
    protected api: MiddlewareAPI;
    set Api(value: MiddlewareAPI) {
        this.api = value;
    }
    get DataExplorer () {
        return getDataExplorer(this.api.getState(), this.Id);
    }
}