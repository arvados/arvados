// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import * as Adapter from 'enzyme-adapter-react-16';

import DataExplorer from "./data-explorer";
import ContextMenu from "../context-menu/context-menu";
import ColumnSelector from "../column-selector/column-selector";
import DataTable from "../data-table/data-table";

configure({ adapter: new Adapter() });

describe("<DataExplorer />", () => {
    it("communicates with <ContextMenu/>", () => {
        const onContextAction = jest.fn();
        const dataExplorer = mount(<DataExplorer
            contextActions={[]}
            onContextAction={onContextAction}
            items={["Item 1"]}
            columns={[{ name: "Column 1", render: jest.fn(), selected: true }]}
            onColumnToggle={jest.fn()}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onSortToggle={jest.fn()} />);

        expect(dataExplorer.find(ContextMenu).prop("actions")).toEqual([]);
        dataExplorer.setState({ contextMenu: { item: "Item 1" } });
        dataExplorer.find(ContextMenu).prop("onActionClick")({ name: "Action 1", icon: "" });
        expect(onContextAction).toHaveBeenCalledWith({ name: "Action 1", icon: "" }, "Item 1");
    });
    
    it("communicates with <ColumnSelector/>", () => {
        const onColumnToggle = jest.fn();
        const columns = [{ name: "Column 1", render: jest.fn(), selected: true }];
        const dataExplorer = mount(<DataExplorer
            columns={columns}
            onColumnToggle={onColumnToggle}
            contextActions={[]}
            onContextAction={jest.fn()}
            items={["Item 1"]}
            onFiltersChange={jest.fn()}
            onRowClick={jest.fn()}
            onSortToggle={jest.fn()} />);

        expect(dataExplorer.find(ColumnSelector).prop("columns")).toBe(columns);
        dataExplorer.find(ColumnSelector).prop("onColumnToggle")("columns");
        expect(onColumnToggle).toHaveBeenCalledWith("columns");
    });
    
    it("communicates with <DataTable/>", () => {
        const onFiltersChange = jest.fn();
        const onSortToggle = jest.fn();
        const onRowClick = jest.fn();
        const columns = [{ name: "Column 1", render: jest.fn(), selected: true }];
        const items = ["Item 1"];
        const dataExplorer = mount(<DataExplorer
            columns={columns}
            items={items}
            onFiltersChange={onFiltersChange}
            onSortToggle={onSortToggle}
            onRowClick={onRowClick}
            onColumnToggle={jest.fn()}
            contextActions={[]}
            onContextAction={jest.fn()} />);

        expect(dataExplorer.find(DataTable).prop("columns")).toBe(columns);
        expect(dataExplorer.find(DataTable).prop("items")).toBe(items);
        dataExplorer.find(DataTable).prop("onRowClick")("event", "rowClick");
        dataExplorer.find(DataTable).prop("onFiltersChange")("filtersChange");
        dataExplorer.find(DataTable).prop("onSortToggle")("sortToggle");
        expect(onFiltersChange).toHaveBeenCalledWith("filtersChange");
        expect(onSortToggle).toHaveBeenCalledWith("sortToggle");
        expect(onRowClick).toHaveBeenCalledWith("rowClick");
    });
});