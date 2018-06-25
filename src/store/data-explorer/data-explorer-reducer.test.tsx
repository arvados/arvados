// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import dataExplorerReducer, { initialDataExplorer } from "./data-explorer-reducer";
import actions from "./data-explorer-action";
import { DataColumn } from "../../components/data-table/data-column";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";

describe('data-explorer-reducer', () => {
    it('should set columns', () => {
        const columns: Array<DataColumn<any>> = [{
            name: "Column 1",
            render: jest.fn(),
            selected: true
        }];
        const state = dataExplorerReducer(undefined,
            actions.SET_COLUMNS({ id: "Data explorer", columns }));
        expect(state["Data explorer"].columns).toEqual(columns);
    });

    it('should toggle sorting', () => {
        const columns: Array<DataColumn<any>> = [{
            name: "Column 1",
            render: jest.fn(),
            selected: true,
            sortDirection: "asc"
        }, {
            name: "Column 2",
            render: jest.fn(),
            selected: true,
            sortDirection: "none",
        }];
        const state = dataExplorerReducer({ "Data explorer": { ...initialDataExplorer, columns } },
            actions.TOGGLE_SORT({ id: "Data explorer", columnName: "Column 2" }));
        expect(state["Data explorer"].columns[0].sortDirection).toEqual("none");
        expect(state["Data explorer"].columns[1].sortDirection).toEqual("asc");
    });

    it('should set filters', () => {
        const columns: Array<DataColumn<any>> = [{
            name: "Column 1",
            render: jest.fn(),
            selected: true,
        }];

        const filters: DataTableFilterItem[] = [{
            name: "Filter 1",
            selected: true
        }];
        const state = dataExplorerReducer({ "Data explorer": { ...initialDataExplorer, columns } },
            actions.SET_FILTERS({ id: "Data explorer", columnName: "Column 1", filters }));
        expect(state["Data explorer"].columns[0].filters).toEqual(filters);
    });

    it('should set items', () => {
        const state = dataExplorerReducer({ "Data explorer": undefined },
            actions.SET_ITEMS({ id: "Data explorer", items: ["Item 1", "Item 2"] }));
        expect(state["Data explorer"].items).toEqual(["Item 1", "Item 2"]);
    });

    it('should set page', () => {
        const state = dataExplorerReducer({ "Data explorer": undefined },
            actions.SET_PAGE({ id: "Data explorer", page: 2 }));
        expect(state["Data explorer"].page).toEqual(2);
    });
    
    it('should set rows per page', () => {
        const state = dataExplorerReducer({ "Data explorer": undefined },
            actions.SET_ROWS_PER_PAGE({ id: "Data explorer", rowsPerPage: 5 }));
        expect(state["Data explorer"].rowsPerPage).toEqual(5);
    });
});
