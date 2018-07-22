// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import { DataTableFilters, DataTableFilterItem } from "./data-table-filters";
import * as Adapter from 'enzyme-adapter-react-16';
import { Checkbox, ButtonBase, ListItem, Button, ListItemText } from "@material-ui/core";

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

    it("updates filters after filters prop change", () => {
        const filters = [{
            name: "Filter 1",
            selected: true
        }];
        const updatedFilters = [, {
            name: "Filter 2",
            selected: true
        }];
        const dataTableFilter = mount(<DataTableFilters name="" filters={filters} />);
        dataTableFilter.find(ButtonBase).simulate("click");
        expect(dataTableFilter.find(Checkbox).prop("checked")).toBeTruthy();
        dataTableFilter.find(ListItem).simulate("click");
        expect(dataTableFilter.find(Checkbox).prop("checked")).toBeFalsy();
        dataTableFilter.setProps({filters: updatedFilters});
        expect(dataTableFilter.find(Checkbox).prop("checked")).toBeTruthy();
        expect(dataTableFilter.find(ListItemText).text()).toBe("Filter 2");
    });

    it("calls onChange with modified list of filters", () => {
        const filters = [{
            name: "Filter 1",
            selected: true
        }, {
            name: "Filter 2",
            selected: false
        }];
        const onChange = jest.fn();
        const dataTableFilter = mount(<DataTableFilters name="" filters={filters} onChange={onChange} />);
        dataTableFilter.find(ButtonBase).simulate("click");
        dataTableFilter.find(ListItem).at(1).simulate("click");
        dataTableFilter.find(Button).at(0).simulate("click");
        expect(onChange).toHaveBeenCalledWith([{
            name: "Filter 1",
            selected: true
        }, {
            name: "Filter 2",
            selected: true
        }]);
    });
});
