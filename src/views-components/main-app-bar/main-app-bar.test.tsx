// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import { MainAppBar, MainAppBarProps } from './main-app-bar';
import { SearchBar } from "~/components/search-bar/search-bar";
import { Breadcrumbs } from "~/components/breadcrumbs/breadcrumbs";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { Button, MenuItem, IconButton } from "@material-ui/core";
import { User } from "~/models/user";
import { MemoryRouter } from 'react-router-dom';

configure({ adapter: new Adapter() });

describe("<MainAppBar />", () => {

    const user: User = {
        firstName: "Test",
        lastName: "User",
        email: "test.user@example.com",
        uuid: "",
        ownerUuid: ""
    };

    it("renders all components and the menu for authenticated user if user prop has value", () => {
        const mainAppBar = mount(
            <MemoryRouter>
                <MainAppBar
                    {...mockMainAppBarProps({ user })}
                />
            </MemoryRouter>
        );
        expect(mainAppBar.find(SearchBar)).toHaveLength(1);
        expect(mainAppBar.find(Breadcrumbs)).toHaveLength(1);
        expect(mainAppBar.find(DropdownMenu)).toHaveLength(2);
    });

    it("renders only the menu for anonymous user if user prop is undefined", () => {
        const menuItems = { accountMenu: [], helpMenu: [], anonymousMenu: [{ label: 'Sign in' }] };
        const mainAppBar = mount(
            <MemoryRouter>
                <MainAppBar
                    {...mockMainAppBarProps({ user: undefined, menuItems })}
                />
            </MemoryRouter>
        );
        expect(mainAppBar.find(SearchBar)).toHaveLength(0);
        expect(mainAppBar.find(Breadcrumbs)).toHaveLength(0);
        expect(mainAppBar.find(DropdownMenu)).toHaveLength(0);
        expect(mainAppBar.find(Button)).toHaveLength(1);
    });

    it("communicates with <SearchBar />", () => {
        const onSearch = jest.fn();
        const mainAppBar = mount(
            <MemoryRouter>
                <MainAppBar
                    {...mockMainAppBarProps({ searchText: 'search text', searchDebounce: 2000, onSearch, user })}
                />
            </MemoryRouter>
        );
        const searchBar = mainAppBar.find(SearchBar);
        expect(searchBar.prop("value")).toBe("search text");
        expect(searchBar.prop("debounce")).toBe(2000);
        searchBar.prop("onSearch")("new search text");
        expect(onSearch).toBeCalledWith("new search text");
    });

    it("communicates with menu", () => {
        const onMenuItemClick = jest.fn();
        const menuItems = { accountMenu: [{ label: "log out" }], helpMenu: [], anonymousMenu: [] };
        const mainAppBar = mount(
            <MemoryRouter>
                <MainAppBar
                    {...mockMainAppBarProps({ menuItems, onMenuItemClick, user })}
                />
            </MemoryRouter>
        );

        mainAppBar.find(DropdownMenu).at(0).find(IconButton).simulate("click");
        mainAppBar.find(DropdownMenu).at(0).find(MenuItem).at(1).simulate("click");
        expect(onMenuItemClick).toBeCalledWith(menuItems.accountMenu[0]);
    });
});

const Breadcrumbs = () => <span>Breadcrumbs</span>;

const mockMainAppBarProps = (props: Partial<MainAppBarProps>): MainAppBarProps => ({
    searchText: '',
    breadcrumbs: Breadcrumbs,
    menuItems: {
        accountMenu: [],
        helpMenu: [],
        anonymousMenu: [],
    },
    buildInfo: '',
    onSearch: jest.fn(),
    onMenuItemClick: jest.fn(),
    onDetailsPanelToggle: jest.fn(),
    ...props,
});
