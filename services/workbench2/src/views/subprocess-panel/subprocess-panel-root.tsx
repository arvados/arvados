// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ContainerRequestState } from 'models/container-request';
import { ResourceKind } from 'models/resource';
import { ProcessIcon } from 'components/icon/icon';
import { SUBPROCESS_PANEL_ID } from 'store/subprocess-panel/subprocess-panel-actions';
import { ResourcesState } from 'store/resources/resources';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Typography } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { Process } from 'store/processes/process';

type CssRules = 'iconHeader' | 'cardHeader';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL,
        marginRight: theme.spacing(2),
    },
    cardHeader: {
        display: 'flex',
        marginTop: '-5px',
        marginBottom: '15px',
    },
});

export interface SubprocessPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export interface SubprocessPanelDataProps {
    process: Process;
    resources: ResourcesState;
}

export interface SubprocessPanelActionProps {
    onRowClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string, resources: ResourcesState) => void;
    onItemDoubleClick: (item: string) => void;
}

type SubprocessPanelProps = SubprocessPanelActionProps & SubprocessPanelDataProps;

const DEFAULT_VIEW_MESSAGES = [
    'No subprocesses available for listing.',
    'The current process may not have any or none matches current filtering.'
];

type SubProcessesTitleProps = WithStyles<CssRules>;

const SubProcessesTitle = withStyles(styles)(
    ({classes}: SubProcessesTitleProps) =>
        <div className={classes.cardHeader}>
            <ProcessIcon className={classes.iconHeader} /><span></span>
            <Typography noWrap variant='h6' color='inherit'>
                Subprocesses
            </Typography>
        </div>
);

export const SubprocessPanelRoot = (props: SubprocessPanelProps & MPVPanelProps) => {
    return <DataExplorer
        id={SUBPROCESS_PANEL_ID}
        onRowClick={props.onRowClick}
        onRowDoubleClick={props.onItemDoubleClick}
        onContextMenu={(event, item) => props.onContextMenu(event, item, props.resources)}
        contextMenuColumn={false}
        defaultViewIcon={ProcessIcon}
        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
        panelName={props.panelName}
        title={<SubProcessesTitle/>}
        parentResource={props.process}
    />;
};
