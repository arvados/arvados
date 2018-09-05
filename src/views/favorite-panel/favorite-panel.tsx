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
import { resourceLabel } from '~/common/labels';
import { ArvadosTheme } from '~/common/custom-theme';
import { FAVORITE_PANEL_ID } from "~/store/favorite-panel/favorite-panel-action";
import {
    ProcessStatus,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceName,
    ResourceOwner,
    ResourceType
} from '~/views-components/data-explorer/renderers';
import { FavoriteIcon } from '~/components/icon/icon';
import { Dispatch } from 'redux';
import { openContextMenu, resourceKindToContextMenuKind } from '~/store/context-menu/context-menu-actions';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { navigateTo } from '~/store/navigation/navigation-action';
import { ContainerRequestState } from "~/models/container-request";
import { FavoritesState } from '../../store/favorites/favorites-reducer';
import { RootState } from '~/store/store';
import { PanelDefaultView } from '~/components/panel-default-view/panel-default-view';

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

export const favoritePanelColumns: DataColumns<string, FavoritePanelFilter> = [
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
        filters: [],
        render: uuid => <ProcessStatus uuid={uuid} />,
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
        render: uuid => <ResourceType uuid={uuid} />,
        width: "125px"
    },
    {
        name: FavoritePanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ResourceOwner uuid={uuid} />,
        width: "200px"
    },
    {
        name: FavoritePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
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
    favorites: FavoritesState;
}

interface FavoritePanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = ({ favorites }: RootState): FavoritePanelDataProps => ({
    favorites
});

const mapDispatchToProps = (dispatch: Dispatch): FavoritePanelActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        const kind = resourceKindToContextMenuKind(resourceUuid);
        if (kind) {
            dispatch<any>(openContextMenu(event, {
                name: '',
                uuid: resourceUuid,
                ownerUuid: '',
                kind: ResourceKind.NONE,
                menuKind: kind
            }));
        }
    },
    onDialogOpen: (ownerUuid: string) => { return; },
    onItemClick: (resourceUuid: string) => {
        dispatch<any>(loadDetailsPanel(resourceUuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateTo(uuid));
    }
});

type FavoritePanelProps = FavoritePanelDataProps & FavoritePanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const FavoritePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<FavoritePanelProps> {
            render() {
                return this.hasAnyFavorites()
                    ? <DataExplorer
                        id={FAVORITE_PANEL_ID}
                        onRowClick={this.props.onItemClick}
                        onRowDoubleClick={this.props.onItemDoubleClick}
                        onContextMenu={this.props.onContextMenu}
                        contextMenuColumn={true} />
                    : <PanelDefaultView
                        icon={FavoriteIcon}
                        messages={['Your favorites list is empty.']} />;
            }

            hasAnyFavorites = () => {
                return Object
                    .keys(this.props.favorites)
                    .find(uuid => this.props.favorites[uuid]);
            }
        }
    )
);
