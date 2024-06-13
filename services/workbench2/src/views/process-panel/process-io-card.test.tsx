// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { mount, configure } from 'enzyme';
import { combineReducers, createStore } from "redux";
import { CircularProgress, MuiThemeProvider, Tab, TableBody } from "@material-ui/core";
import { CustomTheme } from 'common/custom-theme';
import Adapter from "enzyme-adapter-react-16";
import { Provider } from 'react-redux';
import { ProcessIOCard, ProcessIOCardType } from './process-io-card';
import { DefaultView } from "components/default-view/default-view";
import { DefaultVirtualCodeSnippet } from "components/default-code-snippet/default-virtual-code-snippet";
import { ProcessOutputCollectionFiles } from './process-output-collection-files';
import { MemoryRouter } from 'react-router-dom';
import { CopyIcon } from 'components/icon/icon';
import copy from 'copy-to-clipboard';

// Mock collection files component since it just needs to exist
jest.mock('views/process-panel/process-output-collection-files');
// Mock autosizer for the io panel virtual list
jest.mock('react-virtualized-auto-sizer', () => ({ children }: any) => children({ height: 600, width: 600 }));
// Mock copy to clipboard
jest.mock('copy-to-clipboard');

configure({ adapter: new Adapter() });

describe('renderers', () => {
    let store;

    describe('ProcessStatus', () => {

        beforeEach(() => {
            store = createStore(combineReducers({
                auth: (state: any = {}, action: any) => state,
            }));
        });

        it('shows main process input loading when raw or params null', () => {
            // when
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.INPUT}
                            process={false} // Treat as a main process, no requestingContainerUuid
                            params={null}
                            raw={{}}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(Tab).exists()).toBeFalsy();
            expect(panel.find(CircularProgress));

            // when
            panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.INPUT}
                            process={false} // Treat as a main process, no requestingContainerUuid
                            params={[]}
                            raw={null}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(Tab).exists()).toBeFalsy();
            expect(panel.find(CircularProgress));
        });

        it('shows main process empty params and raw', () => {
            // when
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.INPUT}
                            process={false} // Treat as a main process, no requestingContainerUuid
                            params={[]}
                            raw={{}}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).exists()).toBeFalsy();
            expect(panel.find(DefaultView).text()).toEqual('No parameters found');
        });

        it('shows main process with raw and allows copying json', () => {
            // when
            const raw = {some: 'data'};
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.INPUT}
                            process={false} // Treat as a main process, no requestingContainerUuid
                            params={[]}
                            raw={raw}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).length).toBe(1);
            expect(panel.find(DefaultVirtualCodeSnippet).text()).toContain(JSON.stringify(raw, null, 2).replace(/\n/g, ''));

            // when
            panel.find(CopyIcon).simulate('click');
            expect(copy).toHaveBeenCalledWith(JSON.stringify(raw, null, 2), undefined);
        });

        it('shows main process with params', () => {
            // when
            const parameters = [{id: 'someId', label: 'someLabel', value: {display: 'someValue'}}];
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.INPUT}
                            process={false} // Treat as a main process, no requestingContainerUuid
                            params={parameters}
                            raw={{}}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).length).toBe(2); // Empty raw is shown if parameters are present
            expect(panel.find(TableBody).text()).toContain('someId');
            expect(panel.find(TableBody).text()).toContain('someLabel');
            expect(panel.find(TableBody).text()).toContain('someValue');
        });

        it('shows main process with output collection', () => {
            // when
            const outputCollection = '987654321';
            const parameters = [{id: 'someId', label: 'someLabel', value: {display: 'someValue'}}];
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.OUTPUT}
                            process={false} // Treat as a main process, no requestingContainerUuid
                            outputUuid={outputCollection}
                            params={parameters}
                            raw={{}}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).length).toBe(3); // Empty raw is shown if parameters are present
            expect(panel.find(TableBody).text()).toContain('someId');
            expect(panel.find(TableBody).text()).toContain('someLabel');
            expect(panel.find(TableBody).text()).toContain('someValue');

            // Visit output tab
            panel.find(Tab).at(2).simulate('click');
            expect(panel.find(ProcessOutputCollectionFiles).prop('currentItemUuid')).toBe(outputCollection);
        });

        // Subprocess

        it('shows subprocess loading', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.INPUT}
                            process={subprocess} // Treat as a subprocess without outputUuid
                            params={null}
                            raw={null}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(Tab).exists()).toBeFalsy();
            expect(panel.find(CircularProgress));
        });

        it('shows subprocess mounts', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            const sampleMount = {path: '/', pdh: 'abcdef12abcdef12abcdef12abcdef12+0'};
            let panel = mount(
                <Provider store={store}>
                    <MemoryRouter>
                        <MuiThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.INPUT}
                                process={subprocess} // Treat as a subprocess without outputUuid
                                params={null}
                                raw={null}
                                mounts={[sampleMount]}
                            />
                        </MuiThemeProvider>
                    </MemoryRouter>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).length).toBe(1); // Empty raw is hidden in subprocesses
            expect(panel.find(TableBody).text()).toContain(sampleMount.pdh);

        });

        it('shows subprocess output collection', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            const outputCollection = '123456789';
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.OUTPUT}
                            process={subprocess} // Treat as a subprocess with outputUuid
                            outputUuid={outputCollection}
                            params={null}
                            raw={null}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).length).toBe(1); // Unloaded raw is hidden in subprocesses
            expect(panel.find(ProcessOutputCollectionFiles).prop('currentItemUuid')).toBe(outputCollection);
        });

        it('shows empty subprocess raw', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            const outputCollection = '123456789';
            let panel = mount(
                <Provider store={store}>
                    <MuiThemeProvider theme={CustomTheme}>
                        <ProcessIOCard
                            label={ProcessIOCardType.OUTPUT}
                            process={subprocess} // Treat as a subprocess with outputUuid
                            outputUuid={outputCollection}
                            params={null}
                            raw={{}}
                        />
                    </MuiThemeProvider>
                </Provider>
                );

            // then
            expect(panel.find(CircularProgress).exists()).toBeFalsy();
            expect(panel.find(Tab).length).toBe(2); // Empty raw is visible in subprocesses
            expect(panel.find(Tab).first().text()).toBe('Collection');
            expect(panel.find(ProcessOutputCollectionFiles).prop('currentItemUuid')).toBe(outputCollection);
        });

    });
});
