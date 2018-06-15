// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import DataTable from "./data-table";
import { Column } from "./column";
import { TableHead, TableCell, Typography, TableBody, Button } from "@material-ui/core";

configure({ adapter: new Adapter() });

describe("<DataTable />", () => {
    it("shows only selected columns", () => {
        const columns: Array<Column<string>> = [
            {
                header: "Column 1",
                render: () => <span />,
                selected: true
            },
            {
                header: "Column 2",
                render: () => <span />,
                selected: true
            },
            {
                header: "Column 3",
                render: () => <span />,
                selected: false
            }
        ];
        const dataTable = mount(<DataTable columns={columns} items={["item 1"]}/>);
        expect(dataTable.find(TableHead).find(TableCell)).toHaveLength(2);
    });
    
    it("renders header label", () => {
        const columns: Array<Column<string>> = [
            {
                header: "Column 1",
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable columns={columns} items={["item 1"]}/>);
        expect(dataTable.find(TableHead).find(TableCell).text()).toBe("Column 1");
    });
    
    it("uses renderHeader instead of header prop", () => {
        const columns: Array<Column<string>> = [
            {
                header: "Column 1",
                renderHeader: () => <span>Column Header</span>,
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable columns={columns} items={["item 1"]}/>);
        expect(dataTable.find(TableHead).find(TableCell).text()).toBe("Column Header");
    });
    
    it("passes column key prop to corresponding cells", () => {
        const columns: Array<Column<string>> = [
            {
                header: "Column 1",
                key: "column-1-key",
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable columns={columns} items={["item 1"]}/>);
        expect(dataTable.find(TableHead).find(TableCell).key()).toBe("column-1-key");
        expect(dataTable.find(TableBody).find(TableCell).key()).toBe("column-1-key");
    });
    
    it("shows information that items array is empty", () => {
        const columns: Array<Column<string>> = [
            {
                header: "Column 1",
                render: () => <span />,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable columns={columns} items={[]}/>);
        expect(dataTable.find(Typography).text()).toBe("No items");
    });

    it("renders items", () => {
        const columns: Array<Column<string>> = [
            {
                header: "Column 1",
                render: (item) => <Typography>{item}</Typography>,
                selected: true
            },
            {
                header: "Column 2",
                render: (item) => <Button>{item}</Button>,
                selected: true
            }
        ];
        const dataTable = mount(<DataTable columns={columns} items={["item 1"]}/>);
        expect(dataTable.find(TableBody).find(Typography).text()).toBe("item 1");
        expect(dataTable.find(TableBody).find(Button).text()).toBe("item 1");
    });


});