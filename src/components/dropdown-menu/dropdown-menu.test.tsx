// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { shallow, configure } from "enzyme";
import { DropdownMenu } from "./dropdown-menu";
import * as Adapter from 'enzyme-adapter-react-16';
import { MenuItem, IconButton, Menu } from "@material-ui/core";
import { PaginationRightArrowIcon } from "../icon/icon";

configure({ adapter: new Adapter() });

describe("<DropdownMenu />", () => {
    it("renders menu icon", () => {
        const dropdownMenu = shallow(<DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />} />);
        expect(dropdownMenu.find(PaginationRightArrowIcon)).toHaveLength(1);
    });

    it("render menu items", () => {
        const dropdownMenu = shallow(
            <DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />}>
                <MenuItem>Item 1</MenuItem>
                <MenuItem>Item 2</MenuItem>
            </DropdownMenu>
        );
        expect(dropdownMenu.find(MenuItem)).toHaveLength(2);
    });

    it("opens on menu icon click", () => {
        const dropdownMenu = shallow(<DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />} />);
        dropdownMenu.find(IconButton).simulate("click", {currentTarget: {}});
        expect(dropdownMenu.state().anchorEl).toBeDefined();
    });

    it("closes on menu click", () => {
        const dropdownMenu = shallow(<DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />} />);
        dropdownMenu.find(Menu).simulate("click", {currentTarget: {}});
        expect(dropdownMenu.state().anchorEl).toBeUndefined();
    });

});
