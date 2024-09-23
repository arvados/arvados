// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";
import { dataExplorerMiddleware } from "./data-explorer-middleware";
import { dataExplorerActions } from "./data-explorer-action";
import { SortDirection } from "components/data-table/data-column";
import { createTree } from 'models/tree';

describe("DataExplorerMiddleware", () => {

    it("handles only actions that are identified by service id", () => {
        const config = {
            id: "ServiceId",
            columns: [{
                name: "Column",
                selected: true,
                configurable: false,
                sortDirection: SortDirection.NONE,
                filters: createTree(),
                render: cy.stub()
            }],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_PAGE({ id: "OtherId", page: 0 }));
        middleware(dataExplorerActions.SET_PAGE({ id: "ServiceId", page: 0 }));
        middleware(dataExplorerActions.SET_PAGE({ id: "OtherId", page: 0 }));
        expect(api.dispatch).to.be.calledWithMatch(dataExplorerActions.REQUEST_ITEMS({ id: "ServiceId", criteriaChanged: false }));
        expect(api.dispatch).to.be.calledOnce;
    });

    it("handles REQUEST_ITEMS action", () => {
        const config = {
            id: "ServiceId",
            columns: [{
                name: "Column",
                selected: true,
                configurable: false,
                sortDirection: SortDirection.NONE,
                filters: createTree(),
                render: cy.stub()
            }],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.REQUEST_ITEMS({ id: "ServiceId" }));
        expect(api.dispatch).to.be.calledOnce;
    });

    it("handles SET_PAGE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_PAGE({ id: service.getId(), page: 0 }));
        expect(api.dispatch).to.be.calledOnce;
    });

    it("handles SET_ROWS_PER_PAGE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_ROWS_PER_PAGE({ id: service.getId(), rowsPerPage: 0 }));
        expect(api.dispatch).to.be.calledOnce;
    });

    it("handles SET_FILTERS action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_FILTERS({ id: service.getId(), columnName: "", filters: createTree() }));
        expect(api.dispatch).to.be.calledThrice;
    });

    it("handles SET_ROWS_PER_PAGE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_ROWS_PER_PAGE({ id: service.getId(), rowsPerPage: 0 }));
        expect(api.dispatch).to.be.calledOnce;
    });

    it("handles TOGGLE_SORT action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.TOGGLE_SORT({ id: service.getId(), columnName: "" }));
        expect(api.dispatch).to.be.calledOnce;
    });

    it("handles SET_SEARCH_VALUE action", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_EXPLORER_SEARCH_VALUE({ id: service.getId(), searchValue: "" }));
        expect(api.dispatch).to.be.calledThrice;
    });

    it("forwards other actions", () => {
        const config = {
            id: "ServiceId",
            columns: [],
            requestItems: cy.stub(),
            setApi: cy.stub()
        };
        const service = new ServiceMock(config);
        const api = {
            getState: cy.stub(),
            dispatch: cy.stub()
        };
        const next = cy.stub();
        const middleware = dataExplorerMiddleware(service)(api)(next);
        middleware(dataExplorerActions.SET_COLUMNS({ id: service.getId(), columns: [] }));
        middleware(dataExplorerActions.SET_ITEMS({ id: service.getId(), items: [], rowsPerPage: 0, itemsAvailable: 0, page: 0 }));
        middleware(dataExplorerActions.TOGGLE_COLUMN({ id: service.getId(), columnName: "" }));
        expect(api.dispatch).to.not.be.called;
        expect(next).to.be.calledThrice;
    });

});

class ServiceMock extends DataExplorerMiddlewareService {
    constructor(config) {
        super(config.id);
    }

    getColumns() {
        return this.config.columns;
    }

    requestItems(api) {
        this.config.requestItems(api);
        return Promise.resolve();
    }

    async requestCount() {}
}
