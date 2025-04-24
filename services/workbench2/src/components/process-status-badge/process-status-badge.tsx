// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Chip, Button } from '@mui/material';
import { useTheme } from '@mui/material';
import { getProcessStatusStyles } from 'store/processes/process';
import { ProcessStatus } from 'store/processes/process';
import { ArvadosTheme } from 'common/custom-theme';
import { DataTableFilters } from 'components/data-table-filters/data-table-filters';

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

type ProcessStatusSortButtonDataProps = {
    status: string;
    filters: DataTableFilters;
    columnName: string;
    dataExplorerId: string;
    numProcesses: number;
};

type ProcessStatusSortButtonActionProps = {
    onFiltersChange: (filters: DataTableFilters, columnName: string, id: string, status: string) => void;
};

type ProcessStatusSortButtonProps = ProcessStatusSortButtonDataProps & ProcessStatusSortButtonActionProps;

export const ProcessStatusSortButton = ({ status, filters, columnName, dataExplorerId, numProcesses, onFiltersChange }: ProcessStatusSortButtonProps) => {
    const theme = useTheme<ArvadosTheme>();
    const statusText = `${status} (${numProcesses})`;
    return (
        <Button
            data-cy='process-status-chip'
            children={statusText}
            style={{
                height: theme.spacing(3),
                fontSize: '0.875rem',
                borderRadius: theme.spacing(0.625),
                ...getProcessStatusStyles(status, theme),
            }}
            onClick={()=>onFiltersChange(filters, columnName, dataExplorerId, status)}
        />
    );
};

