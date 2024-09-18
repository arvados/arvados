// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import Button, { ButtonProps } from '@mui/material/Button';
import { CircularProgress } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { CircularProgressProps } from '@mui/material/CircularProgress';

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
