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
      data: {
        currentToken: '123123123123',
      },
      open: true,
      dispatch: jest.fn(),
    };
  });

  describe('copy to clipboard', () => {
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
