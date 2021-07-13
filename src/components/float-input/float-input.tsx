// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
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
        const parsedValue = parseFloat(event.target.value).toString();
        event.target.value = parsedValue;
        onChange(event);
    }

    render() {
        const parsedValue = parseFloat(typeof this.props.value === 'string' ? this.props.value : '');
        const value = isNaN(parsedValue) ? '' : parsedValue.toString();
        const props = {
            ...this.props,
            value: value + (this.state.endsWithDecimalSeparator ? '.' : ''),
            onChange: this.handleChange,
        };
        return <Input {...props} />;
    }
}
