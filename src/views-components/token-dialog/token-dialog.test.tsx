// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Button } from '@material-ui/core';
import { mount, configure } from 'enzyme';
import * as Adapter from 'enzyme-adapter-react-16';
import * as CopyToClipboard from 'react-copy-to-clipboard';
import { TokenDialogComponent } from './token-dialog';

configure({ adapter: new Adapter() });

jest.mock('toggle-selection', () => () => () => null);

describe('<CurrentTokenDialog />', () => {
  let props;
  let wrapper;

  beforeEach(() => {
    props = {
      classes: {},
      token: 'xxxtokenxxx',
      apiHost: 'example.com',
      open: true,
      dispatch: jest.fn(),
    };
  });

  describe('Get API Token dialog', () => {
    beforeEach(() => {
      wrapper = mount(<TokenDialogComponent {...props} />);
    });

    it('should include API host and token', () => {
      expect(wrapper.html()).toContain('export ARVADOS_API_HOST=example.com');
      expect(wrapper.html()).toContain('export ARVADOS_API_TOKEN=xxxtokenxxx');
    });

    it('should show the token expiration if present', () => {
      expect(props.tokenExpiration).toBeUndefined();
      expect(wrapper.html()).not.toContain('Expires at:');

      const someDate = '2140-01-01T00:00:00.000Z'
      props.tokenExpiration = new Date(someDate);
      wrapper = mount(<TokenDialogComponent {...props} />);
      expect(wrapper.html()).toContain('Expires at:');
    });

    it('should show a create new token button when allowed', () => {
      expect(props.canCreateNewTokens).toBeFalsy();
      expect(wrapper.html()).not.toContain('GET NEW TOKEN');

      props.canCreateNewTokens = true;
      wrapper = mount(<TokenDialogComponent {...props} />);
      expect(wrapper.html()).toContain('GET NEW TOKEN');
    });
  });

  describe('copy to clipboard button', () => {
    beforeEach(() => {
      wrapper = mount(<TokenDialogComponent {...props} />);
    });

    it('should copy API TOKEN to the clipboard', () => {
      // when
      wrapper.find(CopyToClipboard).find(Button).simulate('click');

      // and
      expect(props.dispatch).toHaveBeenCalledWith({
        payload: {
          hideDuration: 2000,
          kind: 1,
          message: 'Token copied to clipboard',
        },
        type: 'OPEN_SNACKBAR',
      });
    });
  });
});
