import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";
import { dataExplorerMiddleware } from "./data-explorer-middleware";
import { MiddlewareAPI } from "../../../node_modules/redux";
import { columns } from "../../views/project-panel/project-panel";
import { DataColumns } from "../../components/data-table/data-table";
import { dataExplorerActions } from "./data-explorer-action";

// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe("DataExplorerMiddleware", () => {
    it("initializes service with middleware api", () => {
        const config = {
            id: "",
            columns: [],
            requestItems: jest.fn(),
            setApi: jest.fn()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: jest.fn(),
            dispatch: jest.fn()
        };
        dataExplorerMiddleware(service)(api)(jest.fn());
        expect(config.setApi).toHaveBeenCalled();
    });

    it("initializes columns in the store", () => {
        const config = {
            id: "Id",
            columns: [{
                name: "Column",
                selected: true,
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
        dataExplorerMiddleware(service)(api)(next);
        expect(next)
            .toHaveBeenCalledWith(dataExplorerActions.SET_COLUMNS({ id: service.Id, columns: service.Columns }));
    });

    it("handles only actions that are identified by service id", () => {
        const config = {
            id: "ServiceId",
            columns: [{
                name: "Column",
                selected: true,
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
        expect(api.dispatch).toHaveBeenCalledWith(dataExplorerActions.REQUEST_ITEMS({ id: "ServiceId" }));
        expect(api.dispatch).toHaveBeenCalledTimes(1);
    });

    it("handles REQUEST_ITEMS action", () => {
        const config = {
            id: "ServiceId",
            columns: [{
                name: "Column",
                selected: true,
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
        expect(config.requestItems).toHaveBeenCalled();
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
        middleware(dataExplorerActions.SET_PAGE({ id: service.Id, page: 0 }));
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
        middleware(dataExplorerActions.SET_ROWS_PER_PAGE({ id: service.Id, rowsPerPage: 0 }));
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
        middleware(dataExplorerActions.SET_FILTERS({ id: service.Id, columnName: "", filters: [] }));
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
        middleware(dataExplorerActions.SET_PAGE({ id: service.Id, page: 0 }));
        middleware(dataExplorerActions.SET_ROWS_PER_PAGE({ id: service.Id, rowsPerPage: 0 }));
        middleware(dataExplorerActions.SET_FILTERS({ id: service.Id, columnName: "", filters: [] }));
        middleware(dataExplorerActions.TOGGLE_SORT({ id: service.Id, columnName: "" }));
        middleware(dataExplorerActions.TOGGLE_COLUMN({ id: service.Id, columnName: "" }));
        middleware(dataExplorerActions.REQUEST_ITEMS({ id: service.Id }));
        middleware(dataExplorerActions.SET_SEARCH_VALUE({ id: service.Id, searchValue: "" }));
        middleware(dataExplorerActions.RESET_PAGINATION({ id: service.Id }));
        expect(api.dispatch).toHaveBeenCalledTimes(7);
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
        middleware(dataExplorerActions.TOGGLE_SORT({ id: service.Id, columnName: "" }));
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
        middleware(dataExplorerActions.SET_SEARCH_VALUE({ id: service.Id, searchValue: "" }));
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
        middleware(dataExplorerActions.SET_COLUMNS({ id: service.Id, columns: [] }));
        middleware(dataExplorerActions.SET_ITEMS({ id: service.Id, items: [], rowsPerPage: 0, itemsAvailable: 0, page: 0 }));
        middleware(dataExplorerActions.TOGGLE_COLUMN({ id: service.Id, columnName: "" }));
        expect(api.dispatch).toHaveBeenCalledTimes(0);
        expect(next).toHaveBeenCalledTimes(4);
    });

});

class ServiceMock extends DataExplorerMiddlewareService {
    constructor(private config: {
        id: string,
        columns: DataColumns<any>,
        requestItems: (api: MiddlewareAPI) => void;
        setApi: () => void;
    }) {
        super();
    }

    get Id() {
        return this.config.id;
    }

    get Columns() {
        return this.config.columns;
    }

    requestItems() {
        this.config.requestItems(this.api);
    }

    set Api(value: MiddlewareAPI) {
        this.config.setApi();
        this.api = value;
    }

}