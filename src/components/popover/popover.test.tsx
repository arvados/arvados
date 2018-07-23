// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";

import { Popover, DefaultTrigger } from "./popover";
import Button, { ButtonProps } from "@material-ui/core/Button";

configure({ adapter: new Adapter() });

describe("<Popover />", () => {
    it("opens on default trigger click", () => {
        const popover = mount(<Popover />);
        popover.find(DefaultTrigger).simulate("click");
        expect(popover.state().anchorEl).toBeDefined();
    });

    it("renders custom trigger", () => {
        const popover = mount(<Popover triggerComponent={CustomTrigger} />);
        expect(popover.find(Button).text()).toBe("Open popover");
    });

    it("opens on custom trigger click", () => {
        const popover = mount(<Popover triggerComponent={CustomTrigger} />);
        popover.find(CustomTrigger).simulate("click");
        expect(popover.state().anchorEl).toBeDefined();
    });

    it("renders children when opened", () => {
        const popover = mount(
            <Popover>
                <CustomTrigger />
            </Popover>
        );
        popover.find(DefaultTrigger).simulate("click");
        expect(popover.find(CustomTrigger)).toHaveLength(1);
    });

    it("does not close if closeOnContentClick is not set", () => {
        const popover = mount(
            <Popover>
                <CustomTrigger />
            </Popover>
        );
        popover.find(DefaultTrigger).simulate("click");
        popover.find(CustomTrigger).simulate("click");
        expect(popover.state().anchorEl).toBeDefined();
    });
    it("closes on content click if closeOnContentClick is set", () => {
        const popover = mount(
            <Popover closeOnContentClick>
                <CustomTrigger />
            </Popover>
        );
        popover.find(DefaultTrigger).simulate("click");
        popover.find(CustomTrigger).simulate("click");
        expect(popover.state().anchorEl).toBeUndefined();
    });

});

const CustomTrigger: React.SFC<ButtonProps> = (props) => (
    <Button {...props}>
        Open popover
    </Button>
);
