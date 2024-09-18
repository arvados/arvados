// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Typography, Button } from "@mui/material";
import { DataTable } from "./data-table";
import { SortDirection, createDataColumn } from "./data-column";
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from "common/custom-theme";

describe("<DataTable />", () => {
    it("shows only selected columns", () => {
        const columns = [
            createDataColumn({
                name: "Column 1",
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
            createDataColumn({
                name: "Column 2",
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
            createDataColumn({
                name: "Column 3",
                render: () => <span />,
                selected: false,
                configurable: true,
            }),
        ];
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    items={[{ key: "1", name: "item 1" }]}
                    onFiltersChange={cy.stub()}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onContextMenu={cy.stub()}
                    onSortToggle={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        cy.get('th').should('have.length', 3);
    });

    it("renders column name", () => {
        const columns = [
            createDataColumn({
                name: "Column 1",
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
        ];
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    items={["item 1"]}
                    onFiltersChange={cy.stub()}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onContextMenu={cy.stub()}
                    onSortToggle={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        cy.get('th').last().contains('Column 1').should('exist');
    });

    it("uses renderHeader instead of name prop", () => {
        const columns = [
            createDataColumn({
                name: "Column 1",
                renderHeader: () => <span>Column Header</span>,
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
        ];
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    items={[]}
                    onFiltersChange={cy.stub()}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onContextMenu={cy.stub()}
                    onSortToggle={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        cy.get('th').last().contains('Column Header').should('exist');
    });

    it("passes column key prop to corresponding cells", () => {
        const columns = [
            createDataColumn({
                name: "Column 1",
                key: "column-1-key",
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
        ];
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    working={false}
                    items={["item 1"]}
                    onFiltersChange={cy.stub()}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onContextMenu={cy.stub()}
                    onSortToggle={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        setTimeout(() => {
            // cannot access key prop directly, so data-cy is assigned to column.key value
            cy.get('td').last().should('have.attr', 'data-cy', 'column-1-key');
        }, 1000);
    });

    it("renders items", () => {
        const columns = [
            createDataColumn({
                name: "Column 1",
                render: item => <Typography>{item}</Typography>,
                selected: true,
                configurable: true,
            }),
            createDataColumn({
                name: "Column 2",
                render: item => <Button>{item}</Button>,
                selected: true,
                configurable: true,
            }),
        ];
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    working={false}
                    items={["item 1"]}
                    onFiltersChange={cy.stub()}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onContextMenu={cy.stub()}
                    onSortToggle={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        setTimeout(() => {
            cy.get('p').last().contains('item 1').should('exist');
            cy.get('button').last().contains('item 1').should('exist');
        }, 1000);
    });

    it("passes sorting props to <TableSortLabel />", () => {
        const columns = [
            createDataColumn({
                name: "Column 1",
                sort: { direction: SortDirection.ASC, field: "length" },
                selected: true,
                configurable: true,
                render: item => <Typography>{item}</Typography>,
            }),
        ];
        const onSortToggle = cy.spy().as("onSortToggle");
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    items={["item 1"]}
                    onFiltersChange={cy.stub()}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onContextMenu={cy.stub()}
                    onSortToggle={onSortToggle}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        setTimeout(() => {
            cy.get('th').last().contains('Column 1').should('exist');
            cy.get('[data-cy="sort-button"]').should('exist').click();
            cy.get('@onSortToggle').should('have.been.calledWith', columns[1]);
        }, 1000);
    });

    it("does not display <DataTableFiltersPopover /> if there is no filters provided", () => {
        const columns = [
            {
                name: "Column 1",
                selected: true,
                configurable: true,
                filters: [],
                render: item => <Typography>{item}</Typography>,
            },
        ];
        const onFiltersChange = cy.stub();
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    items={[]}
                    onFiltersChange={onFiltersChange}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onSortToggle={cy.stub()}
                    onContextMenu={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />
            </ThemeProvider>
        );
        cy.get('[data-cy=data-table]').should('exist');
        cy.get('[data-cy=popover]').should('not.exist');
    });

    it("passes filter props to <DataTableFiltersPopover />", () => {
        const filters = { Filters : {
            id: 'Filters id',
            active: false,
            children: ['Filter 1', 'Filter 2'],
            expanded: false,
            initialState: true,
            parent: "",
            selected: false,
            status: "LOADED",
            value: { name: 'Filter'}
        } };
        const columns = [
            {
                name: "Column 1",
                selected: true,
                configurable: true,
                filters: filters,
                render: item => <Typography>{item}</Typography>,
            },
        ];
        const onFiltersChange = cy.spy().as("onFiltersChange");
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTable
                    columns={columns}
                    items={[]}
                    onFiltersChange={onFiltersChange}
                    onRowClick={cy.stub()}
                    onRowDoubleClick={cy.stub()}
                    onSortToggle={cy.stub()}
                    onContextMenu={cy.stub()}
                    setCheckedListOnStore={cy.stub()}
                />  
            </ThemeProvider>
        );
        setTimeout(() => {
            cy.get('span[role="button"]').contains('Column 1').should('exist').click();
            cy.get('[data-cy="tree-li"]').contains('Filter').should('exist').click();
            cy.get('@onFiltersChange').should('have.been.calledWith', filters, columns[1]);
        }, 1000);
    });
});
