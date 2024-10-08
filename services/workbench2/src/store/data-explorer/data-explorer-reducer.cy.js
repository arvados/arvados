// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dataExplorerReducer, initialDataExplorer } from "./data-explorer-reducer";
import { dataExplorerActions, DataTableRequestState } from "./data-explorer-action";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";
import { DataColumns } from "../../components/data-table/data-table";
import { SortDirection } from "../../components/data-table/data-column";

describe('data-explorer-reducer', () => {
    it('should set columns', () => {
        const columns = [{
            name: "Column 1",
            filters: [],
            render: cy.stub(),
            selected: true,
            configurable: true,
            sort: {direction: SortDirection.NONE, field: "name"}
        }];
        const state = dataExplorerReducer(undefined,
            dataExplorerActions.SET_COLUMNS({ id: "Data explorer", columns }));
        expect(state["Data explorer"].columns).to.equal(columns);
    });

    it('should toggle sorting', () => {
        const columns = [{
            name: "Column 1",
            filters: [],
            render: cy.stub(),
            selected: true,
            sort: {direction: SortDirection.ASC, field: "name"},
            configurable: true
        }, {
            name: "Column 2",
            filters: [],
            render: cy.stub(),
            selected: true,
            configurable: true,
            sort: {direction: SortDirection.NONE, field: "name"},
        }];
        const state = dataExplorerReducer({ "Data explorer": { ...initialDataExplorer, columns } },
            dataExplorerActions.TOGGLE_SORT({ id: "Data explorer", columnName: "Column 2" }));
        expect(state["Data explorer"].columns[0].sort.direction).to.equal("none");
        expect(state["Data explorer"].columns[1].sort.direction).to.equal("asc");
    });

    it('should set filters', () => {
        const columns = [{
            name: "Column 1",
            filters: [],
            render: cy.stub(),
            selected: true,
            configurable: true,
            sort: {direction: SortDirection.NONE, field: "name"}
        }];

        const filters = [{
            name: "Filter 1",
            selected: true
        }];
        const state = dataExplorerReducer({ "Data explorer": { ...initialDataExplorer, columns } },
            dataExplorerActions.SET_FILTERS({ id: "Data explorer", columnName: "Column 1", filters }));
        expect(state["Data explorer"].columns[0].filters).to.equal(filters);
    });

    it('should set items', () => {
        const items = ["Item 1", "Item 2"];
        let state = dataExplorerReducer({},
            dataExplorerActions.SET_REQUEST_STATE({
                id: "Data explorer",
                requestState: DataTableRequestState.PENDING
            }));
        state = dataExplorerReducer(state,
            dataExplorerActions.SET_ITEMS({
                id: "Data explorer",
                items: items,
                page: 0,
                rowsPerPage: 10,
                itemsAvailable: 100
            }));
        expect(state["Data explorer"].items).to.equal(items);
    });

    it('should set page', () => {
        const state = dataExplorerReducer({},
            dataExplorerActions.SET_PAGE({ id: "Data explorer", page: 2 }));
        expect(state["Data explorer"].page).to.equal(2);
    });

    it('should set rows per page', () => {
        const state = dataExplorerReducer({},
            dataExplorerActions.SET_ROWS_PER_PAGE({ id: "Data explorer", rowsPerPage: 5 }));
        expect(state["Data explorer"].rowsPerPage).to.equal(5);
    });
});
