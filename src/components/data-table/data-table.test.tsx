// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import { TableHead, TableCell, Typography, TableBody, Button, TableSortLabel } from "@material-ui/core";
import * as Adapter from "enzyme-adapter-react-16";
import DataTable, { DataColumns, DataItem } from "./data-table";
import DataTableFilters from "../data-table-filters/data-table-filters";
import { SortDirection } from "./data-column";

configure({ adapter: new Adapter() });

export interface MockItem extends DataItem {
    name: string;
}

describe("<DataTable />", () => {
    it("shows only selected columns", () => {
        const columns: DataColumns<MockItem> = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true
            },
            {
                name: "Column 2",
                render: () => <span />,
                selected: true
            },
            {
                name: "Column 3",
                render: () => <span />,
                selected: false
            }
        ];
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={jest.fn()} />);
        expect(dataTable.find(TableHead).find(TableCell)).toHaveLength(2);
    });

    it("renders column name", () => {
        const columns: DataColumns<MockItem> = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={jest.fn()} />);
        expect(dataTable.find(TableHead).find(TableCell).text()).toBe("Column 1");
    });

    it("uses renderHeader instead of name prop", () => {
        const columns: DataColumns<MockItem> = [
            {
                name: "Column 1",
                renderHeader: () => <span>Column Header</span>,
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={jest.fn()} />);
        expect(dataTable.find(TableHead).find(TableCell).text()).toBe("Column Header");
    });

    it("passes column key prop to corresponding cells", () => {
        const columns: DataColumns<MockItem> = [
            {
                name: "Column 1",
                key: "column-1-key",
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={jest.fn()} />);
        expect(dataTable.find(TableHead).find(TableCell).key()).toBe("column-1-key");
        expect(dataTable.find(TableBody).find(TableCell).key()).toBe("column-1-key");
    });

    it("renders items", () => {
        const columns: DataColumns<MockItem> = [
            {
                name: "Column 1",
                render: (item) => <Typography>{item.name}</Typography>,
                selected: true
            },
            {
                name: "Column 2",
                render: (item) => <Button>{item.name}</Button>,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={jest.fn()} />);
        expect(dataTable.find(TableBody).find(Typography).text()).toBe("item 1");
        expect(dataTable.find(TableBody).find(Button).text()).toBe("item 1");
    });

    it("passes sorting props to <TableSortLabel />", () => {
        const columns: DataColumns<MockItem> = [{
            name: "Column 1",
            sortDirection: SortDirection.Asc,
            selected: true,
            render: (item) => <Typography>{item.name}</Typography>
        }];
        const onSortToggle = jest.fn();
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={onSortToggle} />);
        expect(dataTable.find(TableSortLabel).prop("active")).toBeTruthy();
        dataTable.find(TableSortLabel).at(0).simulate("click");
        expect(onSortToggle).toHaveBeenCalledWith(columns[0]);
    });

    it("passes filter props to <DataTableFilter />", () => {
        const columns: DataColumns<MockItem> = [{
            name: "Column 1",
            sortDirection: SortDirection.Asc,
            selected: true,
            filters: [{ name: "Filter 1", selected: true }],
            render: (item) => <Typography>{item.name}</Typography>
        }];
        const onFiltersChange = jest.fn();
        const dataTable = mount(<DataTable
            columns={columns}
            items={[{ key: "1", name: "item 1" }] as MockItem[]}
            onFiltersChange={onFiltersChange}
            onRowClick={jest.fn()}
            onRowContextMenu={jest.fn()}
            onSortToggle={jest.fn()} />);
        expect(dataTable.find(DataTableFilters).prop("filters")).toBe(columns[0].filters);
        dataTable.find(DataTableFilters).prop("onChange")([]);
        expect(onFiltersChange).toHaveBeenCalledWith([], columns[0]);
    });


});