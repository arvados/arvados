// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import Button, { ButtonProps } from '@material-ui/core/Button';
import { CircularProgress, withStyles } from '@material-ui/core';
import { CircularProgressProps } from '@material-ui/core/CircularProgress';

interface ProgressButtonProps extends ButtonProps {
    loading?: boolean;
    progressProps?: CircularProgressProps;
}

export const ProgressButton = ({ loading, progressProps, children, disabled, ...props }: ProgressButtonProps) =>
    <Button {...props} disabled={disabled || loading}>
        {children}
        {loading && <Progress {...progressProps} size={getProgressSize(props.size)} />}
    </Button>;

const Progress = withStyles({
    root: {
        position: 'absolute',
    },
})(CircularProgress);

const getProgressSize = (size?: 'small' | 'medium' | 'large') => {
    switch (size) {
        case 'small':
            return 16;
        case 'large':
            return 24;
        default:
            return 20;
    }
};
