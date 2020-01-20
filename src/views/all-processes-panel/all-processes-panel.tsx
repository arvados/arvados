// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind } from '~/models/resource';
import { ArvadosTheme } from '~/common/custom-theme';
import { ALL_PROCESSES_PANEL_ID } from '~/store/all-processes-panel/all-processes-panel-action';
import {
    ProcessStatus,
    ResourceLastModifiedDate,
    ResourceName,
    ResourceOwner,
    ResourceType
} from '~/views-components/data-explorer/renderers';
import { ProcessIcon } from '~/components/icon/icon';
import { openContextMenu, resourceKindToContextMenuKind } from '~/store/context-menu/context-menu-actions';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { navigateTo } from '~/store/navigation/navigation-action';
import { ContainerRequestState } from "~/models/container-request";
import { RootState } from '~/store/store';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { createTree } from '~/models/tree';
import { getSimpleObjectTypeFilters } from '~/store/resource-type-filters/resource-type-filters';

type CssRules = "toolbar" | "button";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
});

export enum AllProcessesPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    LAST_MODIFIED = "Last modified"
}

export interface AllProcessesPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const allProcessesPanelColumns: DataColumns<string> = [
    {
        name: AllProcessesPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ProcessStatus uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        // TODO: Only filter by process type (main, subprocess)
        filters: getSimpleObjectTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwner uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.DESC,
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

interface AllProcessesPanelDataProps {
    isAdmin: boolean;
}

interface AllProcessesPanelActionProps {
    onItemClick: (item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = (state : RootState): AllProcessesPanelDataProps => ({
    isAdmin: state.auth.user!.isAdmin
});

type AllProcessesPanelProps = AllProcessesPanelDataProps & AllProcessesPanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const AllProcessesPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<AllProcessesPanelProps> {
            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const menuKind = resourceKindToContextMenuKind(resourceUuid, this.props.isAdmin);
                if (menuKind) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: '',
                        uuid: resourceUuid,
                        ownerUuid: '',
                        kind: ResourceKind.NONE,
                        menuKind
                    }));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch(loadDetailsPanel(uuid));
            }

            render() {
                return <DataExplorer
                    id={ALL_PROCESSES_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={true}
                    dataTableDefaultView={
                        <DataTableDefaultView
                            icon={ProcessIcon}
                            messages={['All Processes list empty.']}
                            />
                    } />;
            }
        }
    )
);
