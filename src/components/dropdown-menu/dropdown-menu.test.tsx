// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { shallow, configure } from "enzyme";
import DropdownMenu from "./dropdown-menu";
import ChevronRightIcon from '@material-ui/icons/ChevronRight';

import * as Adapter from 'enzyme-adapter-react-16';
import { MenuItem, IconButton, Menu } from "@material-ui/core";
import IconBase, { IconTypes } from "../icon/icon";

configure({ adapter: new Adapter() });

describe("<DropdownMenu />", () => {
    it("renders menu icon", () => {
        const dropdownMenu = shallow(<DropdownMenu id="test-menu" icon={IconTypes.CHEVRON_RIGHT} />);
        expect(dropdownMenu.find(IconBase)).toHaveLength(1);
    });

    it("render menu items", () => {
        const dropdownMenu = shallow(
            <DropdownMenu id="test-menu" icon={IconTypes.CHEVRON_RIGHT}>
                <MenuItem>Item 1</MenuItem>
                <MenuItem>Item 2</MenuItem>
            </DropdownMenu>
        );
        expect(dropdownMenu.find(MenuItem)).toHaveLength(2);
    });

    it("opens on menu icon click", () => {
        const dropdownMenu = shallow(<DropdownMenu id="test-menu" icon={IconTypes.CHEVRON_RIGHT} />);
        dropdownMenu.find(IconButton).simulate("click", {currentTarget: {}});
        expect(dropdownMenu.state().anchorEl).toBeDefined();
    });
    
    it("closes on menu click", () => {
        const dropdownMenu = shallow(<DropdownMenu id="test-menu" icon={IconTypes.CHEVRON_RIGHT} />);
        dropdownMenu.find(Menu).simulate("click", {currentTarget: {}});
        expect(dropdownMenu.state().anchorEl).toBeUndefined();
    });

});