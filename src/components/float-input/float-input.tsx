// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Input } from '@material-ui/core';
import { InputProps } from '@material-ui/core/Input';

export class FloatInput extends React.Component<InputProps> {
    state = {
        endsWithDecimalSeparator: false,
    };

    handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const { onChange = () => { return; } } = this.props;
        const [, fraction] = event.target.value.split('.');
        this.setState({ endsWithDecimalSeparator: fraction === '' });
        onChange(event);
    }

    render() {
        const props = {
            ...this.props,
            value: this.props.value + (this.state.endsWithDecimalSeparator ? '.' : ''),
            onChange: this.handleChange,
        };
        return <Input {...props} />;
    }
}
