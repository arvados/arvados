// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import Adapter from 'enzyme-adapter-react-16';
import {configure, shallow } from 'enzyme';

import { AccountMenuComponent } from './account-menu';

configure({ adapter: new Adapter() });

describe('<AccountMenu />', () => {
    let props;
    let wrapper;

    beforeEach(() => {
      props = {
        classes: {},
        user: {
            email: 'email@example.com',
            firstName: 'User',
            lastName: 'Test',
            uuid: 'zzzzz-tpzed-testuseruuid',
            ownerUuid: '',
            username: 'testuser',
            prefs: {},
            isAdmin: false,
            isActive: true
        },
        currentRoute: '',
        workbenchURL: '',
        localCluser: 'zzzzz',
        dispatch: jest.fn(),
      };
    });

    describe('Logout Menu Item', () => {
        beforeEach(() => {
            wrapper = shallow(<AccountMenuComponent {...props} />).dive();
        });

        it('should dispatch a logout action when clicked', () => {
            wrapper.find('[data-cy="logout-menuitem"]').simulate('click');
            expect(props.dispatch).toHaveBeenCalledWith({
                payload: {deleteLinkData: true},
                type: 'LOGOUT',
            });
        });
    });
});
