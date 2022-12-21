// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { mount, configure } from 'enzyme';
import { ProcessStatus, ResourceFileSize } from './renderers';
import Adapter from "enzyme-adapter-react-16";
import { Provider } from 'react-redux';
import configureMockStore from 'redux-mock-store'
import { ResourceKind } from '../../models/resource';
import { ContainerRequestState as CR } from '../../models/container-request';
import { ContainerState as C } from '../../models/container';
import { ProcessStatus as PS } from '../../store/processes/process';

const middlewares = [];
const mockStore = configureMockStore(middlewares);

configure({ adapter: new Adapter() });

describe('renderers', () => {
    let props = null;

    describe('ProcessStatus', () => {
        props = {
            uuid: 'zzzzz-xvhdp-zzzzzzzzzzzzzzz',
            theme: {
                customs: {
                    colors: {
                        // Color values are arbitrary, but they should be
                        // representative of the colors used in the UI.
                        green800: 'rgb(0, 255, 0)',
                        red900: 'rgb(255, 0, 0)',
                        orange: 'rgb(240, 173, 78)',
                        grey600: 'rgb(128, 128, 128)',
                    }
                },
                spacing: {
                    unit: 8,
                },
                palette: {
                    common: {
                        white: 'rgb(255, 255, 255)',
                    },
                },
            },
        };

        [
            // CR Status ; Priority ; C Status ; Exit Code ; C RuntimeStatus ; Expected label ; Expected bg color ; Expected fg color
            [CR.COMMITTED, 1, C.RUNNING, null, {}, PS.RUNNING, props.theme.palette.common.white, props.theme.customs.colors.green800],
            [CR.COMMITTED, 1, C.RUNNING, null, { error: 'whoops' }, PS.FAILING, props.theme.palette.common.white, props.theme.customs.colors.red900],
            [CR.COMMITTED, 1, C.RUNNING, null, { warning: 'watch out!' }, PS.WARNING, props.theme.palette.common.white, props.theme.customs.colors.green800],
            [CR.FINAL, 1, C.CANCELLED, null, {}, PS.CANCELLED, props.theme.customs.colors.red900, props.theme.palette.common.white],
            [CR.FINAL, 1, C.COMPLETE, 137, {}, PS.FAILED, props.theme.customs.colors.red900, props.theme.palette.common.white],
            [CR.FINAL, 1, C.COMPLETE, 0, {}, PS.COMPLETED, props.theme.customs.colors.green800, props.theme.palette.common.white],
            [CR.COMMITTED, 0, C.LOCKED, null, {}, PS.ONHOLD, props.theme.customs.colors.grey600, props.theme.palette.common.white],
            [CR.COMMITTED, 0, C.QUEUED, null, {}, PS.ONHOLD, props.theme.customs.colors.grey600, props.theme.palette.common.white],
            [CR.COMMITTED, 1, C.LOCKED, null, {}, PS.QUEUED, props.theme.palette.common.white, props.theme.customs.colors.grey600],
            [CR.COMMITTED, 1, C.QUEUED, null, {}, PS.QUEUED, props.theme.palette.common.white, props.theme.customs.colors.grey600],
        ].forEach(([crState, crPrio, cState, exitCode, rs, eLabel, eColor, tColor]) => {
            it(`should render the state label '${eLabel}' and color '${eColor}' for CR state=${crState}, priority=${crPrio}, C state=${cState}, exitCode=${exitCode} and RuntimeStatus=${JSON.stringify(rs)}`, () => {
                const containerUuid = 'zzzzz-dz642-zzzzzzzzzzzzzzz';
                const store = mockStore({
                    resources: {
                        [props.uuid]: {
                            kind: ResourceKind.CONTAINER_REQUEST,
                            state: crState,
                            containerUuid: containerUuid,
                            priority: crPrio,
                        },
                        [containerUuid]: {
                            kind: ResourceKind.CONTAINER,
                            state: cState,
                            runtimeStatus: rs,
                            exitCode: exitCode,
                        },
                    }
                });

                const wrapper = mount(<Provider store={store}>
                    <ProcessStatus {...props} />
                </Provider>);

                expect(wrapper.text()).toEqual(eLabel);
                expect(getComputedStyle(wrapper.getDOMNode())
                    .getPropertyValue('color')).toEqual(tColor);
                expect(getComputedStyle(wrapper.getDOMNode())
                    .getPropertyValue('background-color')).toEqual(eColor);
            });
        })
    });

    describe('ResourceFileSize', () => {
        beforeEach(() => {
            props = {
                uuid: 'UUID',
            };
        });

        it('should render collection fileSizeTotal', () => {
            // given
            const store = mockStore({
                resources: {
                    [props.uuid]: {
                        kind: ResourceKind.COLLECTION,
                        fileSizeTotal: 100,
                    }
                }
            });

            // when
            const wrapper = mount(<Provider store={store}>
                <ResourceFileSize {...props}></ResourceFileSize>
            </Provider>);

            // then
            expect(wrapper.text()).toContain('100 B');
        });

        it('should render 0 B as file size', () => {
            // given
            const store = mockStore({ resources: {} });

            // when
            const wrapper = mount(<Provider store={store}>
                <ResourceFileSize {...props}></ResourceFileSize>
            </Provider>);

            // then
            expect(wrapper.text()).toContain('0 B');
        });

        it('should render empty string for non collection resource', () => {
            // given
            const store1 = mockStore({
                resources: {
                    [props.uuid]: {
                        kind: ResourceKind.PROJECT,
                    }
                }
            });
            const store2 = mockStore({
                resources: {
                    [props.uuid]: {
                        kind: ResourceKind.PROJECT,
                    }
                }
            });

            // when
            const wrapper1 = mount(<Provider store={store1}>
                <ResourceFileSize {...props}></ResourceFileSize>
            </Provider>);
            const wrapper2 = mount(<Provider store={store2}>
                <ResourceFileSize {...props}></ResourceFileSize>
            </Provider>);

            // then
            expect(wrapper1.text()).toContain('');
            expect(wrapper2.text()).toContain('');
        });
    });
});
