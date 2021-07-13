// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { mount, configure } from "enzyme";
import { SearchInput, DEFAULT_SEARCH_DEBOUNCE } from "./search-input";
import Adapter from 'enzyme-adapter-react-16';

configure({ adapter: new Adapter() });

describe("<SearchInput />", () => {

    jest.useFakeTimers();

    let onSearch: () => void;

    beforeEach(() => {
        onSearch = jest.fn();
    });

    describe("on submit", () => {
        it("calls onSearch with initial value passed via props", () => {
            const searchInput = mount(<SearchInput value="initial value" onSearch={onSearch} />);
            searchInput.find("form").simulate("submit");
            expect(onSearch).toBeCalledWith("initial value");
        });

        it("calls onSearch with current value", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            searchInput.find("form").simulate("submit");
            expect(onSearch).toBeCalledWith("current value");
        });

        it("calls onSearch with new value passed via props", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            searchInput.setProps({value: "new value"});
            searchInput.find("form").simulate("submit");
            expect(onSearch).toBeCalledWith("new value");
        });

        it("cancels timeout set on input value change", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} debounce={1000} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            searchInput.find("form").simulate("submit");
            jest.runTimersToTime(1000);
            expect(onSearch).toHaveBeenCalledTimes(1);
            expect(onSearch).toBeCalledWith("current value");
        });

    });

    describe("on input value change", () => {
        it("calls onSearch after default timeout", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} />);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            expect(onSearch).not.toBeCalled();
            jest.runTimersToTime(DEFAULT_SEARCH_DEBOUNCE);
            expect(onSearch).toBeCalledWith("current value");
        });

        it("calls onSearch after the time specified in props has passed", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} debounce={2000}/>);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            jest.runTimersToTime(1000);
            expect(onSearch).not.toBeCalled();
            jest.runTimersToTime(1000);
            expect(onSearch).toBeCalledWith("current value");
        });

        it("calls onSearch only once after no change happened during the specified time", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} debounce={1000}/>);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            jest.runTimersToTime(500);
            searchInput.find("input").simulate("change", { target: { value: "changed value" } });
            jest.runTimersToTime(1000);
            expect(onSearch).toHaveBeenCalledTimes(1);
        });

        it("calls onSearch again after the specified time has passed since previous call", () => {
            const searchInput = mount(<SearchInput value="" onSearch={onSearch} debounce={1000}/>);
            searchInput.find("input").simulate("change", { target: { value: "current value" } });
            jest.runTimersToTime(500);
            searchInput.find("input").simulate("change", { target: { value: "intermediate value" } });
            jest.runTimersToTime(1000);
            expect(onSearch).toBeCalledWith("intermediate value");
            searchInput.find("input").simulate("change", { target: { value: "latest value" } });
            jest.runTimersToTime(1000);
            expect(onSearch).toBeCalledWith("latest value");
            expect(onSearch).toHaveBeenCalledTimes(2);

        });

    });

});
