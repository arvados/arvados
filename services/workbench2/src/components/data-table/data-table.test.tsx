// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { mount, configure } from "enzyme";
import { pipe } from "lodash/fp";
import { TableHead, TableCell, Typography, TableBody, Button, TableSortLabel } from "@material-ui/core";
import Adapter from "enzyme-adapter-react-16";
import { DataTable, DataColumns } from "./data-table";
import { SortDirection, createDataColumn } from "./data-column";
import { DataTableFiltersPopover } from "components/data-table-filters/data-table-filters-popover";
import { createTree, setNode, initTreeNode } from "models/tree";
import { DataTableFilterItem } from "components/data-table-filters/data-table-filters-tree";

configure({ adapter: new Adapter() });

describe("<DataTable />", () => {
    it("shows only selected columns", () => {
        const columns: DataColumns<string, string> = [
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
        const dataTable = mount(
            <DataTable
                columns={columns}
                items={[{ key: "1", name: "item 1" }]}
                onFiltersChange={jest.fn()}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onContextMenu={jest.fn()}
                onSortToggle={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        expect(dataTable.find(TableHead).find(TableCell)).toHaveLength(3);
    });

    it("renders column name", () => {
        const columns: DataColumns<string, string> = [
            createDataColumn({
                name: "Column 1",
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
        ];
        const dataTable = mount(
            <DataTable
                columns={columns}
                items={["item 1"]}
                onFiltersChange={jest.fn()}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onContextMenu={jest.fn()}
                onSortToggle={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        expect(dataTable.find(TableHead).find(TableCell).last().text()).toBe("Column 1");
    });

    it("uses renderHeader instead of name prop", () => {
        const columns: DataColumns<string, string> = [
            createDataColumn({
                name: "Column 1",
                renderHeader: () => <span>Column Header</span>,
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
        ];
        const dataTable = mount(
            <DataTable
                columns={columns}
                items={[]}
                onFiltersChange={jest.fn()}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onContextMenu={jest.fn()}
                onSortToggle={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        expect(dataTable.find(TableHead).find(TableCell).last().text()).toBe("Column Header");
    });

    it("passes column key prop to corresponding cells", () => {
        const columns: DataColumns<string, string> = [
            createDataColumn({
                name: "Column 1",
                key: "column-1-key",
                render: () => <span />,
                selected: true,
                configurable: true,
            }),
        ];
        const dataTable = mount(
            <DataTable
                columns={columns}
                working={false}
                items={["item 1"]}
                onFiltersChange={jest.fn()}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onContextMenu={jest.fn()}
                onSortToggle={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        setTimeout(() => {
            expect(dataTable.find(TableBody).find(TableCell).last().key()).toBe("column-1-key");
        }, 1000);
    });

    it("renders items", () => {
        const columns: DataColumns<string, string> = [
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
        const dataTable = mount(
            <DataTable
                columns={columns}
                working={false}
                items={["item 1"]}
                onFiltersChange={jest.fn()}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onContextMenu={jest.fn()}
                onSortToggle={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        setTimeout(() => {
            expect(dataTable.find(TableBody).find(Typography).last().text()).toBe("item 1");
            expect(dataTable.find(TableBody).find(Button).last().text()).toBe("item 1");
        }, 1000);
    });

    it("passes sorting props to <TableSortLabel />", () => {
        const columns: DataColumns<string, string> = [
            createDataColumn({
                name: "Column 1",
                sort: { direction: SortDirection.ASC, field: "length" },
                selected: true,
                configurable: true,
                render: item => <Typography>{item}</Typography>,
            }),
        ];
        const onSortToggle = jest.fn();
        const dataTable = mount(
            <DataTable
                columns={columns}
                items={["item 1"]}
                onFiltersChange={jest.fn()}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onContextMenu={jest.fn()}
                onSortToggle={onSortToggle}
                setCheckedListOnStore={jest.fn()}
            />
        );
        expect(dataTable.find(TableSortLabel).prop("active")).toBeTruthy();
        dataTable.find(TableSortLabel).at(0).simulate("click");
        expect(onSortToggle).toHaveBeenCalledWith(columns[1]);
    });

    it("does not display <DataTableFiltersPopover /> if there is no filters provided", () => {
        const columns: DataColumns<string, string> = [
            {
                name: "Column 1",
                selected: true,
                configurable: true,
                filters: [],
                render: item => <Typography>{item}</Typography>,
            },
        ];
        const onFiltersChange = jest.fn();
        const dataTable = mount(
            <DataTable
                columns={columns}
                items={[]}
                onFiltersChange={onFiltersChange}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onSortToggle={jest.fn()}
                onContextMenu={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        expect(dataTable.find(DataTableFiltersPopover)).toHaveLength(0);
    });

    it("passes filter props to <DataTableFiltersPopover />", () => {
        const filters = pipe(() => createTree<DataTableFilterItem>(), setNode(initTreeNode({ id: "filter", value: { name: "filter" } })));
        const columns: DataColumns<string, string> = [
            {
                name: "Column 1",
                selected: true,
                configurable: true,
                filters: filters(),
                render: item => <Typography>{item}</Typography>,
            },
        ];
        const onFiltersChange = jest.fn();
        const dataTable = mount(
            <DataTable
                columns={columns}
                items={[]}
                onFiltersChange={onFiltersChange}
                onRowClick={jest.fn()}
                onRowDoubleClick={jest.fn()}
                onSortToggle={jest.fn()}
                onContextMenu={jest.fn()}
                setCheckedListOnStore={jest.fn()}
            />
        );
        expect(dataTable.find(DataTableFiltersPopover).prop("filters")).toBe(columns[1].filters);
        dataTable.find(DataTableFiltersPopover).prop("onChange")([]);
        expect(onFiltersChange).toHaveBeenCalledWith([], columns[1]);
    });
});
