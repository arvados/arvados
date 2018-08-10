// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { FavoritePanelItem } from './favorite-panel-item';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "../../views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '../../components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '../../store/store';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '../../models/container-request';
import { SortDirection } from '../../components/data-table/data-column';
import { ResourceKind } from '../../models/resource';
import { resourceLabel } from '../../common/labels';
import { ArvadosTheme } from '../../common/custom-theme';
import { renderName, renderStatus, renderType, renderOwner, renderFileSize, renderDate } from '../../views-components/data-explorer/renderers';
import { FAVORITE_PANEL_ID } from "../../store/favorite-panel/favorite-panel-action";
import { FavoriteIcon } from '../../components/icon/icon';

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

export const columns: DataColumns<FavoritePanelItem, FavoritePanelFilter> = [
    {
        name: FavoritePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        render: renderName,
        width: "450px"
    },
    {
        name: "Status",
        selected: true,
        configurable: true,
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
        render: renderStatus,
        width: "75px"
    },
    {
        name: FavoritePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
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
        render: item => renderType(item.kind),
        width: "125px"
    },
    {
        name: FavoritePanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        render: item => renderOwner(item.owner),
        width: "200px"
    },
    {
        name: FavoritePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        render: item => renderFileSize(item.fileSize),
        width: "50px"
    },
    {
        name: FavoritePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        render: item => renderDate(item.lastModified),
        width: "150px"
    }
];

interface FavoritePanelDataProps {
    currentItemId: string;
}

interface FavoritePanelActionProps {
    onItemClick: (item: FavoritePanelItem) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: FavoritePanelItem) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: FavoritePanelItem) => void;
    onItemRouteChange: (itemId: string) => void;
}

type FavoritePanelProps = FavoritePanelDataProps & FavoritePanelActionProps & DispatchProp
                        & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const FavoritePanel = withStyles(styles)(
    connect((state: RootState) => ({ currentItemId: state.projects.currentItemId }))(
        class extends React.Component<FavoritePanelProps> {
            render() {
                return <DataExplorer
                    id={FAVORITE_PANEL_ID}
                    columns={columns}
                    onRowClick={this.props.onItemClick}
                    onRowDoubleClick={this.props.onItemDoubleClick}
                    onContextMenu={this.props.onContextMenu}
                    extractKey={(item: FavoritePanelItem) => item.uuid} 
                    defaultIcon={FavoriteIcon}
                    defaultMessages={['Your favorites list is empty.']}/>
                ;
            }

            componentWillReceiveProps({ match, currentItemId, onItemRouteChange }: FavoritePanelProps) {
                if (match.params.id !== currentItemId) {
                    onItemRouteChange(match.params.id);
                }
            }
        }
    )
);
