// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { mount, configure } from "enzyme";
import { ConditionalTabs, TabData } from "./conditional-tabs";
import Adapter from 'enzyme-adapter-react-16';
import { Tab } from "@material-ui/core";

configure({ adapter: new Adapter() });

describe("<ConditionalTabs />", () => {
    let tabs: TabData[] = [];

    beforeEach(() => {
        tabs = [{
            show: true,
            label: "Tab1",
            content: <div id="content">Content1</div>,
        },{
            show: false,
            label: "Tab2",
            content: <div id="content">Content2</div>,
        },{
            show: true,
            label: "Tab3",
            content: <div id="content">Content3</div>,
        }];
    });

    it("renders visible tabs", () => {
        // given
        const tabContainer = mount(<ConditionalTabs
            tabs={tabs}
        />);

        // expect 2 visible tabs
        expect(tabContainer.find(Tab)).toHaveLength(2);
        expect(tabContainer.find(Tab).at(0).text()).toBe("Tab1");
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab3");
        expect(tabContainer.find('div#content').text()).toBe("Content1");

        // Show second tab
        tabs[1].show = true;
        tabContainer.setProps({ tabs: tabs });

        // Expect 3 visible tabs
        expect(tabContainer.find(Tab)).toHaveLength(3);
        expect(tabContainer.find(Tab).at(0).text()).toBe("Tab1");
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab2");
        expect(tabContainer.find(Tab).at(2).text()).toBe("Tab3");
        expect(tabContainer.find('div#content').text()).toBe("Content1");
    });

    it("resets selected tab on tab visibility change", () => {
        // given
        const tabContainer = mount(<ConditionalTabs
            tabs={tabs}
        />);

        // Expext second tab to be Tab3
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab3");
        // Click on Tab3
        tabContainer.find(Tab).at(1).simulate('click');
        expect(tabContainer.find('div#content').text()).toBe("Content3");

        // when Tab2 becomes visible
        tabs[1].show = true;
        tabContainer.setProps({ tabs: tabs });

        // Selected tab resets
        expect(tabContainer.find('div#content').text()).toBe("Content1");
    });
});
