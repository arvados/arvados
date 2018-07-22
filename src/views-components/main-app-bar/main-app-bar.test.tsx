// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure, ReactWrapper } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import { MainAppBar } from "./main-app-bar";
import { SearchBar } from "../../components/search-bar/search-bar";
import { Breadcrumbs } from "../../components/breadcrumbs/breadcrumbs";
import { DropdownMenu } from "../../components/dropdown-menu/dropdown-menu";
import { Button, MenuItem, IconButton } from "@material-ui/core";
import { User } from "../../models/user";

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
            <MainAppBar
                user={user}
                onContextMenu={jest.fn()}
                onDetailsPanelToggle={jest.fn()}
                onContextMenu={jest.fn()}
                {...{ searchText: "", breadcrumbs: [], menuItems: { accountMenu: [], helpMenu: [], anonymousMenu: [] }, onSearch: jest.fn(), onBreadcrumbClick: jest.fn(), onMenuItemClick: jest.fn() }}
            />
        );
        expect(mainAppBar.find(SearchBar)).toHaveLength(1);
        expect(mainAppBar.find(Breadcrumbs)).toHaveLength(1);
        expect(mainAppBar.find(DropdownMenu)).toHaveLength(2);
    });

    it("renders only the menu for anonymous user if user prop is undefined", () => {
        const menuItems = { accountMenu: [], helpMenu: [], anonymousMenu: [{ label: 'Sign in' }] };
        const mainAppBar = mount(
            <MainAppBar
                menuItems={menuItems}
                onDetailsPanelToggle={jest.fn()}
                onContextMenu={jest.fn()}
                {...{ searchText: "", breadcrumbs: [], onSearch: jest.fn(), onBreadcrumbClick: jest.fn(), onMenuItemClick: jest.fn() }}
            />
        );
        expect(mainAppBar.find(SearchBar)).toHaveLength(0);
        expect(mainAppBar.find(Breadcrumbs)).toHaveLength(0);
        expect(mainAppBar.find(DropdownMenu)).toHaveLength(0);
        expect(mainAppBar.find(Button)).toHaveLength(1);
    });

    it("communicates with <SearchBar />", () => {
        const onSearch = jest.fn();
        const mainAppBar = mount(
            <MainAppBar
                searchText="search text"
                searchDebounce={2000}
                onContextMenu={jest.fn()}
                onSearch={onSearch}
                onContextMenu={jest.fn()}
                onDetailsPanelToggle={jest.fn()}
                {...{ user, breadcrumbs: [], menuItems: { accountMenu: [], helpMenu: [], anonymousMenu: [] }, onBreadcrumbClick: jest.fn(), onMenuItemClick: jest.fn() }}
            />
        );
        const searchBar = mainAppBar.find(SearchBar);
        expect(searchBar.prop("value")).toBe("search text");
        expect(searchBar.prop("debounce")).toBe(2000);
        searchBar.prop("onSearch")("new search text");
        expect(onSearch).toBeCalledWith("new search text");
    });

    it("communicates with <Breadcrumbs />", () => {
        const items = [{ label: "breadcrumb 1" }];
        const onBreadcrumbClick = jest.fn();
        const mainAppBar = mount(
            <MainAppBar
                breadcrumbs={items}
                onContextMenu={jest.fn()}
                onBreadcrumbClick={onBreadcrumbClick}
                onContextMenu={jest.fn()}
                onDetailsPanelToggle={jest.fn()}
                {...{ user, searchText: "", menuItems: { accountMenu: [], helpMenu: [], anonymousMenu: [] }, onSearch: jest.fn(), onMenuItemClick: jest.fn() }}
            />
        );
        const breadcrumbs = mainAppBar.find(Breadcrumbs);
        expect(breadcrumbs.prop("items")).toBe(items);
        breadcrumbs.prop("onClick")(items[0]);
        expect(onBreadcrumbClick).toBeCalledWith(items[0]);
    });

    it("communicates with menu", () => {
        const onMenuItemClick = jest.fn();
        const menuItems = { accountMenu: [{label: "log out"}], helpMenu: [], anonymousMenu: [] };
        const mainAppBar = mount(
            <MainAppBar
                menuItems={menuItems}
                onContextMenu={jest.fn()}
                onMenuItemClick={onMenuItemClick}
                onContextMenu={jest.fn()}
                onDetailsPanelToggle={jest.fn()}
                {...{ user, searchText: "", breadcrumbs: [], onSearch: jest.fn(), onBreadcrumbClick: jest.fn() }}
            />
        );

        mainAppBar.find(DropdownMenu).at(0).find(IconButton).simulate("click");
        mainAppBar.find(DropdownMenu).at(0).find(MenuItem).at(1).simulate("click");
        expect(onMenuItemClick).toBeCalledWith(menuItems.accountMenu[0]);
    });
});
