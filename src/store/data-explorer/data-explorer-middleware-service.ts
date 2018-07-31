// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { DataColumns } from "../../components/data-table/data-table";
import { RootState } from "../store";

export abstract class DataExplorerMiddlewareService {
    protected readonly id: string;

    protected constructor(id: string) {
        this.id = id;
    }

    public getId() {
        return this.id;
    }

    abstract getColumns(): DataColumns<any>;
    abstract requestItems(api: MiddlewareAPI<Dispatch, RootState>): void;
}
