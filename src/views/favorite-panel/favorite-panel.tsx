// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '~/store/store';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '~/models/container-request';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { ArvadosTheme } from '~/common/custom-theme';
import { FAVORITE_PANEL_ID, loadFavoritePanel } from "~/store/favorite-panel/favorite-panel-action";
import { ResourceFileSize, ResourceLastModifiedDate, ProcessStatus, ResourceType, ResourceOwner, ResourceName } from '~/views-components/data-explorer/renderers';
import { FavoriteIcon } from '~/components/icon/icon';
import { Dispatch } from 'redux';
import { contextMenuActions } from '~/store/context-menu/context-menu-actions';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { loadDetailsPanel } from '../../store/details-panel/details-panel-action';
import { navigateToResource } from '~/store/navigation/navigation-action';

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

export enum FavoritePanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export interface FavoritePanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const columns: DataColumns<string, FavoritePanelFilter> = [
    {
        name: FavoritePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: uuid => <ResourceName uuid={uuid} />,
        width: "450px"
    },
    {
        name: "Status",
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [
            {
                name: ContainerRequestState.COMMITTED,
                selected: true,
                type: ContainerRequestState.COMMITTED
            },
            {
                name: ContainerRequestState.FINAL,
                selected: true,
                type: ContainerRequestState.FINAL
            },
            {
                name: ContainerRequestState.UNCOMMITTED,
                selected: true,
                type: ContainerRequestState.UNCOMMITTED
            }
        ],
        render: uuid => <ProcessStatus uuid={uuid} />,
        width: "75px"
    },
    {
        name: FavoritePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [
            {
                name: resourceLabel(ResourceKind.COLLECTION),
                selected: true,
                type: ResourceKind.COLLECTION
            },
            {
                name: resourceLabel(ResourceKind.PROCESS),
                selected: true,
                type: ResourceKind.PROCESS
            },
            {
                name: resourceLabel(ResourceKind.PROJECT),
                selected: true,
                type: ResourceKind.PROJECT
            }
        ],
        render: uuid => <ResourceType uuid={uuid} />,
        width: "125px"
    },
    {
        name: FavoritePanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceOwner uuid={uuid} />,
        width: "200px"
    },
    {
        name: FavoritePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceFileSize uuid={uuid} />,
        width: "50px"
    },
    {
        name: FavoritePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />,
        width: "150px"
    }
];

interface FavoritePanelDataProps {
    currentItemId: string;
}

interface FavoritePanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
    onMount: () => void;
}

const mapDispatchToProps = (dispatch: Dispatch): FavoritePanelActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        event.preventDefault();
        dispatch(
            contextMenuActions.OPEN_CONTEXT_MENU({
                position: { x: event.clientX, y: event.clientY },
                resource: { name: '', uuid: resourceUuid, kind: ContextMenuKind.RESOURCE }
            })
        );
    },
    onDialogOpen: (ownerUuid: string) => { return; },
    onItemClick: (resourceUuid: string) => {
        dispatch<any>(loadDetailsPanel(resourceUuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateToResource(uuid));
    },
    onMount: () => {
        dispatch(loadFavoritePanel());
    },
});

type FavoritePanelProps = FavoritePanelDataProps & FavoritePanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const FavoritePanel = withStyles(styles)(
    connect((state: RootState) => ({ currentItemId: state.projects.currentItemId }), mapDispatchToProps)(
        class extends React.Component<FavoritePanelProps> {
            render() {
                return <DataExplorer
                    id={FAVORITE_PANEL_ID}
                    columns={columns}
                    onRowClick={this.props.onItemClick}
                    onRowDoubleClick={this.props.onItemDoubleClick}
                    onContextMenu={this.props.onContextMenu}
                    defaultIcon={FavoriteIcon}
                    defaultMessages={['Your favorites list is empty.']} />;
            }

            componentDidMount() {
                this.props.onMount();
            }
        }
    )
);
