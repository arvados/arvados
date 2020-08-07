// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { shallow, configure } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import * as CopyToClipboard from "react-copy-to-clipboard";
import { CurrentTokenDialogComponent } from "./current-token-dialog";

configure({ adapter: new Adapter() });

describe("<CurrentTokenDialog />", () => {
  let props;
  let wrapper;

  beforeEach(() => {
    props = {
      classes: {},
      data: {
        currentToken: "123123123123",
      },
      dispatch: jest.fn(),
    };
  });

  describe("copy to clipboard", () => {
    beforeEach(() => {
      wrapper = shallow(<CurrentTokenDialogComponent {...props} />);
    });

    it("should copy API TOKEN to the clipboard", () => {
      // when
      wrapper.find(CopyToClipboard).props().onCopy();

      // then
      expect(props.dispatch).toHaveBeenCalledWith({
        payload: {
          hideDuration: 2000,
          kind: 1,
          message: "Token copied to clipboard",
        },
        type: "OPEN_SNACKBAR",
      });
    });
  });
});
