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
            content: <div id="content1">Content1</div>,
        },{
            show: false,
            label: "Tab2",
            content: <div id="content2">Content2</div>,
        },{
            show: true,
            label: "Tab3",
            content: <div id="content3">Content3</div>,
        }];
    });

    it("renders only visible tabs", () => {
        // given
        const tabContainer = mount(<ConditionalTabs
            tabs={tabs}
        />);

        // expect 2 visible tabs
        expect(tabContainer.find(Tab)).toHaveLength(2);
        expect(tabContainer.find(Tab).at(0).text()).toBe("Tab1");
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab3");
        // expect visible content 1 and tab 3 to be hidden but exist
        // content 2 stays unrendered since the tab is hidden
        expect(tabContainer.find('div#content1').text()).toBe("Content1");
        expect(tabContainer.find('div#content1').prop('hidden')).toBeFalsy();
        expect(tabContainer.find('div#content2').exists()).toBeFalsy();
        expect(tabContainer.find('div#content3').prop('hidden')).toBeTruthy();

        // Show second tab
        tabs[1].show = true;
        tabContainer.setProps({ tabs: tabs });
        tabContainer.update();

        // Expect 3 visible tabs
        expect(tabContainer.find(Tab)).toHaveLength(3);
        expect(tabContainer.find(Tab).at(0).text()).toBe("Tab1");
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab2");
        expect(tabContainer.find(Tab).at(2).text()).toBe("Tab3");
        // Expect visible content 1 and hidden content 2/3
        expect(tabContainer.find('div#content1').text()).toBe("Content1");
        expect(tabContainer.find('div#content1').prop('hidden')).toBeFalsy();
        expect(tabContainer.find('div#content2').prop('hidden')).toBeTruthy();
        expect(tabContainer.find('div#content3').prop('hidden')).toBeTruthy();

        // Click on Tab2 (position 1)
        tabContainer.find(Tab).at(1).simulate('click');

        // Expect 3 visible tabs
        expect(tabContainer.find(Tab)).toHaveLength(3);
        expect(tabContainer.find(Tab).at(0).text()).toBe("Tab1");
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab2");
        expect(tabContainer.find(Tab).at(2).text()).toBe("Tab3");
        // Expect visible content2 and hidden content 1/3
        expect(tabContainer.find('div#content2').text()).toBe("Content2");
        expect(tabContainer.find('div#content1').prop('hidden')).toBeTruthy();
        expect(tabContainer.find('div#content2').prop('hidden')).toBeFalsy();
        expect(tabContainer.find('div#content3').prop('hidden')).toBeTruthy();
    });

    it("resets selected tab on tab visibility change", () => {
        // given
        const tabContainer = mount(<ConditionalTabs
            tabs={tabs}
        />);

        // Expect second tab to be Tab3
        expect(tabContainer.find(Tab).at(1).text()).toBe("Tab3");
        // Click on Tab3 (position 2)
        tabContainer.find(Tab).at(1).simulate('click');
        expect(tabContainer.find('div#content3').text()).toBe("Content3");
        expect(tabContainer.find('div#content1').prop('hidden')).toBeTruthy();
        expect(tabContainer.find('div#content2').exists()).toBeFalsy();
        expect(tabContainer.find('div#content3').prop('hidden')).toBeFalsy();

        // when Tab2 becomes visible
        tabs[1].show = true;
        tabContainer.setProps({ tabs: tabs });
        tabContainer.update(); // Needed or else tab1 content will still be hidden

        // Selected tab resets to 1, tabs 2/3 are hidden
        expect(tabContainer.find('div#content1').text()).toBe("Content1");
        expect(tabContainer.find('div#content1').prop('hidden')).toBeFalsy();
        expect(tabContainer.find('div#content2').prop('hidden')).toBeTruthy();
        expect(tabContainer.find('div#content3').prop('hidden')).toBeTruthy();
    });
});
