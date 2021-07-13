// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";
import { dataExplorerMiddleware } from "./data-explorer-middleware";
import { MiddlewareAPI } from "redux";
import { DataColumns } from "components/data-table/data-table";
import { dataExplorerActions } from "./data-explorer-action";
import { SortDirection } from "components/data-table/data-column";
import { createTree } from 'models/tree';
import { DataTableFilterItem } from "components/data-table-filters/data-table-filters-tree";


describe("DataExplorerMiddleware", () => {

    it("handles only actions that are identified by service id", () => {
        const config = {
            id: "ServiceId",
            columns: [{
                name: "Column",
                selected: true,
                configurable: false,
                sortDirection: SortDirection.NONE,
                filters: createTree<DataTableFilterItem>(),
                render: jest.fn()
            }],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_PAGE({ id: "OtherId", page: 0 }));
        middleware(dataExplorerActions.SET_PAGE({ id: "ServiceId", page: 0 }));
        middleware(dataExplorerActions.SET_PAGE({ id: "OtherId", page: 0 }));
        expect(api.dispatch).toHaveBeenCalledWith(dataExplorerActions.REQUEST_ITEMS({ id: "ServiceId", criteriaChanged: false }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles REQUEST_ITEMS action", () => {
        const config = {
            id: "ServiceId",
            columns: [{
                name: "Column",
                selected: true,
                configurable: false,
                sortDirection: SortDirection.NONE,
                filters: createTree<DataTableFilterItem>(),
                render: jest.fn()
            }],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.REQUEST_ITEMS({ id: "ServiceId" }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles SET_PAGE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_PAGE({ id: service.getId(), page: 0 }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles SET_ROWS_PER_PAGE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_ROWS_PER_PAGE({ id: service.getId(), rowsPerPage: 0 }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles SET_FILTERS action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_FILTERS({ id: service.getId(), columnName: "", filters: createTree() }));
        expect(api.dispatch).toHaveBeenCalledTimes(2);
    });

    it("handles SET_ROWS_PER_PAGE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_ROWS_PER_PAGE({ id: service.getId(), rowsPerPage: 0 }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles TOGGLE_SORT action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.TOGGLE_SORT({ id: service.getId(), columnName: "" }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles SET_SEARCH_VALUE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_EXPLORER_SEARCH_VALUE({ id: service.getId(), searchValue: "" }));
        expect(api.dispatch).toHaveBeenCalledTimes(2);
    });

    it("forwards other actions", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        const next = jest.fn();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_COLUMNS({ id: service.getId(), columns: [] }));
        middleware(dataExplorerActions.SET_ITEMS({ id: service.getId(), items: [], rowsPerPage: 0, itemsAvailable: 0, page: 0 }));
        middleware(dataExplorerActions.TOGGLE_COLUMN({ id: service.getId(), columnName: "" }));
        expect(api.dispatch).toHaveBeenCalledTimes(0);
        expect(next).toHaveBeenCalledTimes(3);
    });

});

class ServiceMock extends DataExplorerMiddlewareService {
    constructor(private config: {
        id: string,
        columns: DataColumns<any>,
        requestItems: (api: MiddlewareAPI) => Promise<void>
    }) {
        super(config.id);
    }

    getColumns() {
        return this.config.columns;
    }

    requestItems(api: MiddlewareAPI): Promise<void> {
        this.config.requestItems(api);
        return Promise.resolve();
    }
}
