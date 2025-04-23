// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Chip } from '@mui/material';
import { useTheme } from '@mui/material';
import { getProcessStatusStyles } from 'store/processes/process';
import { ProcessStatus } from 'store/processes/process';
import { ArvadosTheme } from 'common/custom-theme';

export const ProcessStatusBadge = ({ status }: { status: ProcessStatus }) => {
    const theme = useTheme<ArvadosTheme>();
    return (
        <Chip
            data-cy='process-status-chip'
            label={status}
            style={{
                height: theme.spacing(3),
                width: theme.spacing(12),
                fontSize: '0.875rem',
                borderRadius: theme.spacing(0.625),
                ...getProcessStatusStyles(status, theme),
            }}
        />
    );
};
