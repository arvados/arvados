// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import copy from 'copy-to-clipboard';

interface CopyToClipboardProps {
  getText: (() => string);
  children: any
  onCopy?(text: string, result: boolean): void;
  options?: {
    debug?: boolean;
    message?: string;
    format?: string; // MIME type
  };
}

export default class CopyResultToClipboard extends React.PureComponent<CopyToClipboardProps> {
  static defaultProps = {
    onCopy: undefined,
    options: undefined
  };

  onClick = event => {
    const {
      getText,
      onCopy,
      children,
      options
    } = this.props;

    const elem = React.Children.only(children);

    const text = getText();

    const result = copy(text, options);

    if (onCopy) {
      onCopy(text, result);
    }

    // Bypass onClick if it was present
    if (elem && elem.props && typeof elem.props.onClick === 'function') {
      elem.props.onClick(event);
    }
  };


  render() {
    const {
      getText: _getText,
      onCopy: _onCopy,
      options: _options,
      children,
      ...props
    } = this.props;
    const elem = React.Children.only(children);

    return React.cloneElement(elem, {...props, onClick: this.onClick});
  }
}
