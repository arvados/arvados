// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";

import { DataExplorer } from "./data-explorer";
import { DataTableFetchMode } from "../data-table/data-table";
import { ProjectIcon } from "../icon/icon";
import { SortDirection } from "../data-table/data-column";
import { combineReducers, createStore } from "redux";
import { Provider } from "react-redux";
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from "common/custom-theme";

describe("<DataExplorer />", () => {
    let store;
    beforeEach(() => {
        const initialMSState = {
            multiselect: {
                checkedList: {},
                isVisible: false,
            },
            resources: {},
        };
        store = createStore(
            combineReducers({
                multiselect: (state = initialMSState.multiselect, action) => state,
                resources: (state = initialMSState.resources, action) => state,
            })
        );
    });

    it("communicates with <SearchInput/>", () => {
        const onSearch = cy.stub().as("onSearch");
        const onSetColumns = cy.stub();

        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <DataExplorer
                    {...mockDataExplorerProps()}
                    items={[{ name: "item 1" }]}
                    searchValue="search value"
                    onSearch={onSearch}
                    onSetColumns={onSetColumns}
                />
              </ThemeProvider>
            </Provider>
        );
        cy.get('input[type=text]').should('have.value', 'search value');
        cy.get('input[type=text]').clear();
        cy.get('input[type=text]').type('new value');
        cy.get('@onSearch').should('have.been.calledWith', 'new value');
    });

    it("communicates with <ColumnSelector/>", () => {
        const onColumnToggle = cy.spy().as("onColumnToggle");
        const onSetColumns = cy.stub();
        const columns = [{ name: "Column 1", render: cy.stub(), selected: true, configurable: true, sortDirection: SortDirection.ASC, filters: {}}];
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <DataExplorer
                    {...mockDataExplorerProps()}
                    columns={columns}
                    onColumnToggle={onColumnToggle}
                    items={[{ name: "item 1" }]}
                    onSetColumns={onSetColumns}
                />
              </ThemeProvider>
            </Provider>
        );
        cy.get('[data-cy=column-selector-button]').should('exist').click();
        cy.get('[data-cy=column-selector-li]').contains('Column 1').should('exist').click();
        cy.get('@onColumnToggle').should('have.been.calledWith', columns[0]);
    });

    it("communicates with <DataTable/>", () => {
        const onFiltersChange = cy.spy().as("onFiltersChange");
        const onSortToggle = cy.spy().as("onSortToggle");
        const onRowClick = cy.spy().as("onRowClick");
        const onSetColumns = cy.stub();
        const filters = { Filters : {
            id: 'Filters id',
            active: false,
            children: ['Filter 1', 'Filter 2'],
            expanded: false,
            initialState: true,
            parent: "",
            selected: false,
            status: "LOADED",
            value: { name: 'Filters'}
        } };
        const columns = [
            { name: "Column 1", render: cy.stub(), selected: true, configurable: true, sortDirection: SortDirection.ASC, filters },
            { name: "Column 2", render: cy.stub(), selected: true, configurable: true, sortDirection: SortDirection.ASC, filters: {}, sort: true }
        ];
        const items = [{ name: "item 1" }];
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <DataExplorer
                    {...mockDataExplorerProps()}
                    columns={columns}
                    items={items}
                    onFiltersChange={onFiltersChange}
                    onSortToggle={onSortToggle}
                    onRowClick={onRowClick}
                    onSetColumns={onSetColumns}
                />
              </ThemeProvider>
            </Provider>
        );
        //check if the table and column are rendered
        cy.get('[data-cy=data-table]').should('exist');
        cy.get('[data-cy=data-table]').contains('Column 1').should('exist');
        //check onRowClick
        cy.get('[data-cy=data-table-row]').should('exist');
        cy.get('[data-cy=data-table-row]').click();
        cy.get('@onRowClick').should('have.been.calledWith', items[0]);
        //check onFiltersChange
        cy.contains('Column 1').click();
        cy.get('[data-cy=tree-li]').contains('Filters').click();
        cy.get('@onFiltersChange').should('have.been.calledWith', filters, columns[0] );
        cy.contains('Close').click();
        //check onSortToggle
        cy.contains('Column 2').click();
        cy.get('@onSortToggle').should('have.been.calledWith', columns[1]);
    });

    it("communicates with <TablePagination/>", () => {
        const onPageChange = cy.spy().as("onPageChange");
        const onChangeRowsPerPage = cy.spy().as("onChangeRowsPerPage");
        const onSetColumns = cy.stub();
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <DataExplorer
                    {...mockDataExplorerProps()}
                    items={hundredItems}
                    itemsAvailable={100}
                    page={0}
                    rowsPerPage={50}
                    rowsPerPageOptions={[10, 20, 50, 100]}
                    onPageChange={onPageChange}
                    onChangeRowsPerPage={onChangeRowsPerPage}
                    onSetColumns={onSetColumns}
                />
              </ThemeProvider>
            </Provider>
        );
        //check if the pagination is rendered
        cy.get('[data-cy=table-pagination]').should('exist');
        cy.get('[data-cy=table-pagination]').contains('1–50 of 100').should('exist');
        cy.get('p').contains('Rows per page:').should('exist');
        //check onPageChange
        cy.get('button[title="Go to next page"]').should('exist').click();
        cy.get('@onPageChange').should('have.been.calledWith', 1);
        //check onChangeRowsPerPage
        cy.get('input[value=50]').should('exist').parent().click();
        cy.get('li[data-value=10]').should('exist').click();
        cy.get('@onChangeRowsPerPage').should('have.been.calledWith', 10);
    });
});

const mockDataExplorerProps = () => ({
    fetchMode: DataTableFetchMode.PAGINATED,
    columns: [],
    items: [],
    itemsAvailable: 0,
    contextActions: [],
    searchValue: "",
    page: 0,
    rowsPerPage: 0,
    rowsPerPageOptions: [0],
    onSearch: cy.stub(),
    onFiltersChange: cy.stub(),
    onSortToggle: cy.stub(),
    onRowClick: cy.stub(),
    onRowDoubleClick: cy.stub(),
    onColumnToggle: cy.stub(),
    onPageChange: cy.stub(),
    onChangeRowsPerPage: cy.stub(),
    onContextMenu: cy.stub(),
    defaultIcon: ProjectIcon,
    onSetColumns: cy.stub(),
    onLoadMore: cy.stub(),
    defaultMessages: ["testing"],
    contextMenuColumn: true,
    setCheckedListOnStore: cy.stub(),
    toggleMSToolbar: cy.stub(),
    isMSToolbarVisible: false,
    checkedList: {},
});

const hundredItems = Array.from({ length: 100 }, (v, i) => ({ name: `item ${i}` }));