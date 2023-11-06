// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// This mocks react-copy-to-clipboard's dependency module to avoid warnings
// from jest when running tests. As we're not testing copy-to-clipboard, it's
// safe to just mock it.
// https://github.com/nkbt/react-copy-to-clipboard/issues/106#issuecomment-605227151
jest.mock('copy-to-clipboard', () => {
  return jest.fn();
});

import React from 'react';
import { Button } from '@material-ui/core';
import { mount, configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import CopyToClipboard from 'react-copy-to-clipboard';
import { TokenDialogComponent } from './token-dialog';
import { combineReducers, createStore } from 'redux';
import { Provider } from 'react-redux';

configure({ adapter: new Adapter() });

jest.mock('toggle-selection', () => () => () => null);

describe('<CurrentTokenDialog />', () => {
  let props;
  let wrapper;
  let store;

  beforeEach(() => {
    props = {
      classes: {},
      token: 'xxxtokenxxx',
      apiHost: 'example.com',
      open: true,
      dispatch: jest.fn(),
    };

    const initialAuthState = {
      localCluster: "zzzzz",
      remoteHostsConfig: {},
      sessions: {},
    };

    store = createStore(combineReducers({
      auth: (state: any = initialAuthState, action: any) => state,
    }));
  });

  describe('Get API Token dialog', () => {
    beforeEach(() => {
      wrapper = mount(
        <Provider store={store}>
          <TokenDialogComponent {...props} />
        </Provider>
      );
    });

    it('should include API host and token', () => {
      expect(wrapper.html()).toContain('export ARVADOS_API_HOST=example.com');
      expect(wrapper.html()).toContain('export ARVADOS_API_TOKEN=xxxtokenxxx');
    });

    it('should show the token expiration if present', () => {
      expect(props.tokenExpiration).toBeUndefined();
      expect(wrapper.html()).toContain('This token does not have an expiration date');

      const someDate = '2140-01-01T00:00:00.000Z'
      props.tokenExpiration = new Date(someDate);
      wrapper = mount(
        <Provider store={store}>
          <TokenDialogComponent {...props} />
        </Provider>);
      expect(wrapper.html()).toContain(props.tokenExpiration.toLocaleString());
    });

    it('should show a create new token button when allowed', () => {
      expect(props.canCreateNewTokens).toBeFalsy();
      expect(wrapper.html()).not.toContain('GET NEW TOKEN');

      props.canCreateNewTokens = true;
      wrapper = mount(
        <Provider store={store}>
          <TokenDialogComponent {...props} />
        </Provider>);
      expect(wrapper.html()).toContain('GET NEW TOKEN');
    });
  });

  describe('copy to clipboard button', () => {
    beforeEach(() => {
      wrapper = mount(
        <Provider store={store}>
          <TokenDialogComponent {...props} />
        </Provider>);
    });

    it('should copy API TOKEN to the clipboard', () => {
      // when
      wrapper.find(CopyToClipboard).find(Button).simulate('click');

      // and
      expect(props.dispatch).toHaveBeenCalledWith({
        payload: {
          hideDuration: 2000,
          kind: 1,
          message: 'Shell code block copied',
        },
        type: 'OPEN_SNACKBAR',
      });
    });
  });
});
