// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import { DataTableFiltersPopover } from "./data-table-filters-popover";
import * as Adapter from 'enzyme-adapter-react-16';
import { Checkbox, IconButton } from "@material-ui/core";
import { getInitialProcessStatusFilters } from "store/resource-type-filters/resource-type-filters"

configure({ adapter: new Adapter() });

describe("<DataTableFiltersPopover />", () => {
    it("renders filters according to their state", () => {
        // 1st filter (All) is selected, the rest aren't.
        const filters = getInitialProcessStatusFilters()

        const dataTableFilter = mount(<DataTableFiltersPopover name="" filters={filters} />);
        dataTableFilter.find(IconButton).simulate("click");
        expect(dataTableFilter.find(Checkbox).at(0).prop("checked")).toBeTruthy();
        expect(dataTableFilter.find(Checkbox).at(1).prop("checked")).toBeFalsy();
    });
});
