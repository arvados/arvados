// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataColumns } from 'components/data-table/data-table';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ContainerRequestState } from 'models/container-request';
import { SortDirection } from 'components/data-table/data-column';
import { ResourceKind } from 'models/resource';
import { ResourceCreatedAtDate, ProcessStatus, ContainerRunTime } from 'views-components/data-explorer/renderers';
import { ProcessIcon } from 'components/icon/icon';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { SUBPROCESS_PANEL_ID } from 'store/subprocess-panel/subprocess-panel-actions';
import { createTree } from 'models/tree';
import { getInitialProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
import { ResourcesState } from 'store/resources/resources';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { StyleRulesCallback, Typography, WithStyles, withStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { ProcessResource } from 'models/process';
import { SubprocessProgressBar } from 'components/subprocess-progress-bar/subprocess-progress-bar';
import { Process } from 'store/processes/process';

type CssRules = 'iconHeader' | 'cardHeader';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL,
        marginRight: theme.spacing.unit * 2,
    },
    cardHeader: {
        display: 'flex',
    },
});

export enum SubprocessPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    CREATED_AT = "Created At",
    RUNTIME = "Run Time"
}

export interface SubprocessPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const subprocessPanelColumns: DataColumns<string, ProcessResource> = [
    {
        name: SubprocessPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "name"},
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: SubprocessPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: uuid => <ProcessStatus uuid={uuid} />,
    },
    {
        name: SubprocessPanelColumnNames.CREATED_AT,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.DESC, field: "createdAt"},
        filters: createTree(),
        render: uuid => <ResourceCreatedAtDate uuid={uuid} />
    },
    {
        name: SubprocessPanelColumnNames.RUNTIME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ContainerRunTime uuid={uuid} />
    }
];

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
        contextMenuColumn={true}
        defaultViewIcon={ProcessIcon}
        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
        doHidePanel={props.doHidePanel}
        doMaximizePanel={props.doMaximizePanel}
        doUnMaximizePanel={props.doUnMaximizePanel}
        panelMaximized={props.panelMaximized}
        panelName={props.panelName}
        title={<SubProcessesTitle/>}
        progressBar={<SubprocessProgressBar parentResource={props.process} />}
        forceMultiSelectMode
    />;
};
