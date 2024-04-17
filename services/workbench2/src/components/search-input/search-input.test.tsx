// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { mount, configure } from "enzyme";
import { SearchInput, DEFAULT_SEARCH_DEBOUNCE } from "./search-input";
import Adapter from 'enzyme-adapter-react-16';

configure({ adapter: new Adapter() });

describe("<SearchInput />", () => {

    // jest.useFakeTimers() applies to all setTimeout functions
    jest.useFakeTimers();

    let onSearch: () => void;

    beforeEach(() => {
        onSearch = jest.fn();
    });

    describe("on submit", () => {
        it("calls onSearch with initial value passed via props", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="initial value" onSearch={onSearch} />);
            searchInput.find("form").simulate("submit");
            expect(onSearch).toBeCalledWith("initial value");
        });

        it("calls onSearch with current value", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            searchInput.find("form").simulate("submit");
            expect(onSearch).toBeCalledWith("current value");
        });

        it("calls onSearch with new value passed via props", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            searchInput.setProps({value: "new value"});
            searchInput.find("form").simulate("submit");
            expect(onSearch).toBeCalledWith("new value");
        });

        it("cancels timeout set on input value change", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={1000} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            searchInput.find("form").simulate("submit");
            jest.runTimersToTime(1000);
            expect(onSearch).toHaveBeenCalledTimes(1);
            expect(onSearch).toBeCalledWith("current value");
        });

    });

    describe("on input value change", () => {
        it("calls onSearch after default timeout", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            expect(onSearch).not.toBeCalled();
            setTimeout(() => {
                expect(onSearch).toBeCalledWith("current value");
            }, DEFAULT_SEARCH_DEBOUNCE);
        });

        it("calls onSearch after the time specified in props has passed", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={2000}/>);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            expect(onSearch).not.toBeCalled();
            setTimeout(() => {
                expect(onSearch).toBeCalledWith("current value");
            }, 1000);
        });

        it("calls onSearch only once after no change happened during the specified time", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={1000}/>);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            setTimeout(() => {
                searchInput.find("input").simulate("change", { target: { value: "changed value" } });
            }, 500);
            setTimeout(() => {
                expect(onSearch).toHaveBeenCalledTimes(1);
            }, 1000);
        });

        it("calls onSearch again after the specified time has passed since previous call", () => {
            const searchInput = mount(<SearchInput selfClearProp="" value="" onSearch={onSearch} debounce={1000}/>);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            setTimeout(() => {
                searchInput.find("input").simulate("change", { target: { value: "intermediate value" } });
            }, 500);
            setTimeout(() => {
                expect(onSearch).toBeCalledWith("intermediate value");
            }, 1000);
            searchInput.find("input").simulate("change", { target: { value: "latest value" } });
            setTimeout(() => {
                expect(onSearch).toBeCalledWith("latest value");
                expect(onSearch).toHaveBeenCalledTimes(2);
            }, 1000);

        });

    });

    describe("on input target change", () => {
        it("clears the input value on selfClearProp change", () => {
            const searchInput = mount(<SearchInput selfClearProp="abc" value="123" onSearch={onSearch} debounce={1000}/>);

            // component should clear value upon creation
            setTimeout(() => {
                expect(onSearch).toBeCalledWith("");
                expect(onSearch).toHaveBeenCalledTimes(1);
            }, 1000);

            // component should not clear on same selfClearProp
            searchInput.setProps({ selfClearProp: 'abc' });
            setTimeout(() => {
                expect(onSearch).toHaveBeenCalledTimes(1);
            }, 1000);

            // component should clear on selfClearProp change
            searchInput.setProps({ selfClearProp: '111' });
            setTimeout(() => {
                expect(onSearch).toBeCalledWith("");
                expect(onSearch).toHaveBeenCalledTimes(2);
            }, 1000);
        });
    });
});
