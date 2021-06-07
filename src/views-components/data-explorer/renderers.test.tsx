// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { mount, configure } from 'enzyme';
import { ResourceFileSize } from './renderers';
import * as Adapter from "enzyme-adapter-react-16";
import { Provider } from 'react-redux';
import configureMockStore from 'redux-mock-store'
import { ResourceKind } from '../../models/resource';

const middlewares = [];
const mockStore = configureMockStore(middlewares);

configure({ adapter: new Adapter() });

describe('renderers', () => {
    let props = null;

    describe('ResourceFileSize', () => {
        beforeEach(() => {
            props = {
                uuid: 'UUID',
            };
        });

        it('should render collection fileSizeTotal', () => {
            // given
            const store = mockStore({ resources: {
                [props.uuid]: {
                    kind: ResourceKind.COLLECTION,
                    fileSizeTotal: 100,
                }
            }});

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
            const store1 = mockStore({ resources: {
                [props.uuid]: {
                    kind: ResourceKind.PROJECT,
                }
            }});
            const store2 = mockStore({ resources: {
                [props.uuid]: {
                    kind: ResourceKind.PROJECT,
                }
            }});

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