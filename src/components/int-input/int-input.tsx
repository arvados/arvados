// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Input } from '@material-ui/core';
import { InputProps } from '@material-ui/core/Input';

export class IntInput extends React.Component<InputProps> {
    handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const { onChange = () => { return; } } = this.props;
        const parsedValue = parseInt(event.target.value, 10);
        event.target.value = parsedValue.toString();
        onChange(event);
    }

    render() {
        const parsedValue = parseInt(typeof this.props.value === 'string' ? this.props.value : '', 10);
        const value = isNaN(parsedValue) ? '' : parsedValue.toString();
        const props = {
            ...this.props,
            value,
            onChange: this.handleChange,
        };
        return <Input {...props} />;
    }
}
