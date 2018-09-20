// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from '~/store/store';
import { WorkflowIcon } from '~/components/icon/icon';
import { ResourcesState, getResource } from '~/store/resources/resources';
import { navigateTo } from "~/store/navigation/navigation-action";
import { loadDetailsPanel } from "~/store/details-panel/details-panel-action";
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { WORKFLOW_PANEL_ID } from '~/store/workflow-panel/workflow-panel-actions';
import { openContextMenu } from '~/store/context-menu/context-menu-actions';
import { GroupResource } from '~/models/group';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import {
    ResourceLastModifiedDate,
    ResourceName,
} from "~/views-components/data-explorer/renderers";
import { SortDirection } from '~/components/data-table/data-column';
import { DataColumns } from '~/components/data-table/data-table';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { Grid } from '@material-ui/core';
import { WorkflowDescriptionCard } from './workflow-description-card';

export enum WorkflowPanelColumnNames {
    NAME = "Name",
    AUTHORISATION = "Authorisation",
    LAST_MODIFIED = "Last modified",
}

interface WorkflowPanelDataProps {
    resources: ResourcesState;
}

export enum ResourceStatus {
    PUBLIC = 'public',
    PRIVATE = 'private',
    SHARED = 'shared'
}

const resourceStatus = (type: string) => {
    switch (type) {
        case ResourceStatus.PUBLIC:
            return "Public";
        case ResourceStatus.PRIVATE:
            return "Private";
        case ResourceStatus.SHARED:
            return "Shared";
        default:
            return "Unknown";
    }
};

export const workflowPanelColumns: DataColumns<string, DataTableFilterItem> = [
    {
        name: WorkflowPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: (uuid: string) => <ResourceName uuid={uuid} />
    },
    {
        name: WorkflowPanelColumnNames.AUTHORISATION,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [
            {
                name: resourceStatus(ResourceStatus.PUBLIC),
                selected: true,
            },
            {
                name: resourceStatus(ResourceStatus.PRIVATE),
                selected: true,
            },
            {
                name: resourceStatus(ResourceStatus.SHARED),
                selected: true,
            }
        ],
        render: (uuid: string) => <ResourceName uuid={uuid} />,
    },
    {
        name: WorkflowPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: (uuid: string) => <ResourceLastModifiedDate uuid={uuid} />
    }
];

type WorkflowPanelProps = WorkflowPanelDataProps & DispatchProp;

export const WorkflowPanel = connect((state: RootState) => ({
    resources: state.resources
}))(
    class extends React.Component<WorkflowPanelProps> {
        render() {
            return <Grid container>
                <Grid item xs={6} style={{ paddingRight: '24px', display: 'grid' }}>
                    <DataExplorer
                        id={WORKFLOW_PANEL_ID}
                        onRowClick={this.handleRowClick}
                        onRowDoubleClick={this.handleRowDoubleClick}
                        onContextMenu={this.handleContextMenu}
                        contextMenuColumn={false}
                        dataTableDefaultView={<DataTableDefaultView icon={WorkflowIcon} />} />
                </Grid>
                <Grid item xs={6}>
                    <WorkflowDescriptionCard />
                </Grid>
            </Grid>;
        }

        handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
            const resource = getResource<GroupResource>(resourceUuid)(this.props.resources);
            if (resource) {
                this.props.dispatch<any>(openContextMenu(event, {
                    name: '',
                    uuid: resource.uuid,
                    ownerUuid: resource.ownerUuid,
                    isTrashed: resource.isTrashed,
                    kind: resource.kind,
                    menuKind: ContextMenuKind.PROJECT,
                }));
            }
        }

        handleRowDoubleClick = (uuid: string) => {
            this.props.dispatch<any>(navigateTo(uuid));
        }

        handleRowClick = (uuid: string) => {
            this.props.dispatch(loadDetailsPanel(uuid));
        }
    }
);