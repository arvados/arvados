// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import { DataTableFilters } from "./data-table-filters";
import * as Adapter from 'enzyme-adapter-react-16';
import { Checkbox, ButtonBase } from "@material-ui/core";

configure({ adapter: new Adapter() });

describe("<DataTableFilter />", () => {
    it("renders filters according to their state", () => {
        const filters = [{
            name: "Filter 1",
            selected: true
        }, {
            name: "Filter 2",
            selected: false
        }];
        const dataTableFilter = mount(<DataTableFilters name="" filters={filters} />);
        dataTableFilter.find(ButtonBase).simulate("click");
        expect(dataTableFilter.find(Checkbox).at(0).prop("checked")).toBeTruthy();
        expect(dataTableFilter.find(Checkbox).at(1).prop("checked")).toBeFalsy();
    });
});
