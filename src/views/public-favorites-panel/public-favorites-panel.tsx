// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from 'components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { SortDirection } from 'components/data-table/data-column';
import { ResourceKind } from 'models/resource';
import { ArvadosTheme } from 'common/custom-theme';
import {
    ProcessStatus,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceType,
    ResourceName,
    ResourceOwner
} from 'views-components/data-explorer/renderers';
import { PublicFavoriteIcon } from 'components/icon/icon';
import { Dispatch } from 'redux';
import {
    openContextMenu,
    resourceUuidToContextMenuKind
} from 'store/context-menu/context-menu-actions';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { navigateTo } from 'store/navigation/navigation-action';
import { ContainerRequestState } from "models/container-request";
import { RootState } from 'store/store';
import { DataTableDefaultView } from 'components/data-table-default-view/data-table-default-view';
import { createTree } from 'models/tree';
import { getSimpleObjectTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { PUBLIC_FAVORITE_PANEL_ID } from 'store/public-favorites-panel/public-favorites-action';
import { PublicFavoritesState } from 'store/public-favorites/public-favorites-reducer';

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

export enum PublicFavoritePanelColumnNames {
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

export const publicFavoritePanelColumns: DataColumns<string> = [
    {
        name: PublicFavoritePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: "Status",
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ProcessStatus uuid={uuid} />
    },
    {
        name: PublicFavoritePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getSimpleObjectTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />
    },
    {
        name: PublicFavoritePanelColumnNames.OWNER,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwner uuid={uuid} />
    },
    {
        name: PublicFavoritePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: PublicFavoritePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.DESC,
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

interface PublicFavoritePanelDataProps {
    publicFavorites: PublicFavoritesState;
}

interface PublicFavoritePanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = ({ publicFavorites }: RootState): PublicFavoritePanelDataProps => ({
    publicFavorites
});

const mapDispatchToProps = (dispatch: Dispatch): PublicFavoritePanelActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        const kind = dispatch<any>(resourceUuidToContextMenuKind(resourceUuid));
        if (kind) {
            dispatch<any>(openContextMenu(event, {
                name: '',
                uuid: resourceUuid,
                ownerUuid: '',
                kind: ResourceKind.NONE,
                menuKind: kind
            }));
        }
        dispatch<any>(loadDetailsPanel(resourceUuid));
    },
    onDialogOpen: (ownerUuid: string) => { return; },
    onItemClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateTo(uuid));
    }
});

type FavoritePanelProps = PublicFavoritePanelDataProps & PublicFavoritePanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const PublicFavoritePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<FavoritePanelProps> {
            render() {
                return <DataExplorer
                    id={PUBLIC_FAVORITE_PANEL_ID}
                    onRowClick={this.props.onItemClick}
                    onRowDoubleClick={this.props.onItemDoubleClick}
                    onContextMenu={this.props.onContextMenu}
                    contextMenuColumn={true}
                    dataTableDefaultView={
                        <DataTableDefaultView
                            icon={PublicFavoriteIcon}
                            messages={['Public favorites list is empty.']} />
                    } />;
            }
        }
    )
);
