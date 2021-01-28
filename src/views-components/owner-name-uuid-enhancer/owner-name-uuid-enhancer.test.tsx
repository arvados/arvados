// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { mount, configure } from 'enzyme';
import * as Adapter from "enzyme-adapter-react-16";
import { OwnerNameUuidEnhancer, OwnerNameUuidEnhancerProps } from './owner-name-uuid-enhancer';

configure({ adapter: new Adapter() });

describe('NotFoundPanelRoot', () => {
    let props: OwnerNameUuidEnhancerProps;

    beforeEach(() => {
        props = {
            ownerNamesMap: {},
            fetchOwner: () => {},
            uuid: 'zzzz-tpzed-xxxxxxxxxxxxxxx',
        };
    });

    it('should render uuid without name', () => {
        // when
        const wrapper = mount(<OwnerNameUuidEnhancer {...props} />);

        // then
        expect(wrapper.html()).toBe('<span>zzzz-tpzed-xxxxxxxxxxxxxxx</span>');
    });

    it('should render uuid with name', () => {
        // given
        const fullName = 'John Doe';

        // setup
        props.ownerNamesMap = {
            [props.uuid]: fullName
        };

        // when
        const wrapper = mount(<OwnerNameUuidEnhancer {...props} />);

        // then
        expect(wrapper.html()).toBe('<span>zzzz-tpzed-xxxxxxxxxxxxxxx (John Doe)</span>');
    });
});