// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { configure, mount } from "enzyme";

import Adapter from "enzyme-adapter-react-16";
import { Breadcrumbs } from "./breadcrumbs";
import { Button, ThemeProvider, Theme, StyledEngineProvider } from "@mui/material";
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import { CustomTheme } from 'common/custom-theme';
import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";


declare module '@mui/styles/defaultTheme' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  interface DefaultTheme extends Theme {}
}


configure({ adapter: new Adapter() });

describe("<Breadcrumbs />", () => {

    let onClick: () => void;
    let resources = {};
    let store;
    beforeEach(() => {
        onClick = jest.fn();
        const initialAuthState = {
            config: {
                clusterConfig: {
                    Collections: {
                        ForwardSlashNameSubstitution: "/"
                    }
                }
            }
        }
        store = createStore(combineReducers({
            auth: (state: any = initialAuthState, action: any) => state,
        }));
    });

    it("renders one item", () => {
        const items = [
            { label: 'breadcrumb 1', uuid: '1' }
        ];
        const breadcrumbs = mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={jest.fn()} />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);
        expect(breadcrumbs.find(Button)).toHaveLength(1);
        expect(breadcrumbs.find(ChevronRightIcon)).toHaveLength(0);
    });

    it("renders multiple items", () => {
        const items = [
            { label: 'breadcrumb 1', uuid: '1' },
            { label: 'breadcrumb 2', uuid: '2' }
        ];
        const breadcrumbs = mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={jest.fn()} />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);
        expect(breadcrumbs.find(Button)).toHaveLength(2);
        expect(breadcrumbs.find(ChevronRightIcon)).toHaveLength(1);
    });

    it("calls onClick with clicked item", () => {
        const items = [
            { label: 'breadcrumb 1', uuid: '1' },
            { label: 'breadcrumb 2', uuid: '2' }
        ];
        const breadcrumbs = mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={jest.fn()} />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);
        breadcrumbs.find(Button).at(1).simulate('click');
        expect(onClick).toHaveBeenCalledWith(expect.any(Function), items[1]);
    });

});
