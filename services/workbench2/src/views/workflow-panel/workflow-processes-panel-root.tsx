// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ContainerRequestState } from 'models/container-request';
import { ResourceKind } from 'models/resource';
import { ProcessIcon } from 'components/icon/icon';
import { WORKFLOW_PROCESSES_PANEL_ID } from 'store/workflow-panel/workflow-panel-actions';
import { ResourcesState } from 'store/resources/resources';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Typography } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { getResource } from 'store/resources/resources';
import { WorkflowResource } from 'models/workflow';
import { RootState } from 'store/store';

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

export interface WorkflowProcessesPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export interface WorkflowProcessesPanelDataProps {
    resources: ResourcesState;
    workflow?: WorkflowResource;
}

export interface WorkflowProcessesPanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string, resources: ResourcesState) => void;
    onItemDoubleClick: (item: string) => void;
}

type WorkflowProcessesPanelProps = WorkflowProcessesPanelActionProps & WorkflowProcessesPanelDataProps;

const DEFAULT_VIEW_MESSAGES = [
    'No processes available for listing.',
    'The current process may not have any or none matches current filtering.'
];

type WorkflowProcessesTitleProps = WithStyles<CssRules>;

const WorkflowProcessesTitle = withStyles(styles)(
    ({ classes }: WorkflowProcessesTitleProps) =>
        <div className={classes.cardHeader}>
            <ProcessIcon className={classes.iconHeader} /><span></span>
            <Typography noWrap variant='h6' color='inherit'>
                Run History
            </Typography>
        </div>
);

const mapStateToProps = (state: RootState): Pick<WorkflowProcessesPanelDataProps, 'workflow'> => {
    const currentRouteUuid = state.properties.currentRouteUuid;
    const workflow = getResource<WorkflowResource>(currentRouteUuid)(state.resources);
    return {
        workflow,
    };
};

export const WorkflowProcessesPanelRoot = connect(mapStateToProps)((props: WorkflowProcessesPanelProps & MPVPanelProps) => {
    return <DataExplorer
        id={WORKFLOW_PROCESSES_PANEL_ID}
        onRowClick={props.onItemClick}
        onRowDoubleClick={props.onItemDoubleClick}
        onContextMenu={(event, item) => props.onContextMenu(event, item, props.resources)}
        contextMenuColumn={false}
        defaultViewIcon={ProcessIcon}
        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
        panelName={props.panelName}
        parentResource={props.workflow}
        title={<WorkflowProcessesTitle />}
        />;
});
