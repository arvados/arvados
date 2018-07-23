// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import { ColumnSelector, ColumnSelectorTrigger } from "./column-selector";
import { ListItem, Checkbox } from "@material-ui/core";
import { DataColumns } from "../data-table/data-table";

configure({ adapter: new Adapter() });

describe("<ColumnSelector />", () => {
    it("shows only configurable columns", () => {
        const columns: DataColumns<void> = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true
            },
            {
                name: "Column 2",
                render: () => <span />,
                selected: true,
                configurable: true,
            },
            {
                name: "Column 3",
                render: () => <span />,
                selected: true,
                configurable: false
            }
        ];
        const columnsConfigurator = mount(<ColumnSelector columns={columns} onColumnToggle={jest.fn()} />);
        columnsConfigurator.find(ColumnSelectorTrigger).simulate("click");
        expect(columnsConfigurator.find(ListItem)).toHaveLength(2);
    });

    it("renders checked checkboxes next to selected columns", () => {
        const columns: DataColumns<void> = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true
            },
            {
                name: "Column 2",
                render: () => <span />,
                selected: false
            },
            {
                name: "Column 3",
                render: () => <span />,
                selected: true
            }
        ];
        const columnsConfigurator = mount(<ColumnSelector columns={columns} onColumnToggle={jest.fn()} />);
        columnsConfigurator.find(ColumnSelectorTrigger).simulate("click");
        expect(columnsConfigurator.find(Checkbox).at(0).prop("checked")).toBe(true);
        expect(columnsConfigurator.find(Checkbox).at(1).prop("checked")).toBe(false);
        expect(columnsConfigurator.find(Checkbox).at(2).prop("checked")).toBe(true);
    });

    it("calls onColumnToggle with clicked column", () => {
        const columns: DataColumns<void> = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true
            }
        ];
        const onColumnToggle = jest.fn();
        const columnsConfigurator = mount(<ColumnSelector columns={columns} onColumnToggle={onColumnToggle} />);
        columnsConfigurator.find(ColumnSelectorTrigger).simulate("click");
        columnsConfigurator.find(ListItem).simulate("click");
        expect(onColumnToggle).toHaveBeenCalledWith(columns[0]);
    });
});
