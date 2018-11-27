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
import { FavoritesState } from '~/store/favorites/favorites-reducer';
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

export const favoritePanelColumns: DataColumns<string> = [
    {
        name: FavoritePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
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
        name: FavoritePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getSimpleObjectTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />
    },
    {
        name: FavoritePanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwner uuid={uuid} />
    },
    {
        name: FavoritePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: FavoritePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
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
        dispatch<any>(loadDetailsPanel(resourceUuid));
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
                return <DataExplorer
                    id={FAVORITE_PANEL_ID}
                    onRowClick={this.props.onItemClick}
                    onRowDoubleClick={this.props.onItemDoubleClick}
                    onContextMenu={this.props.onContextMenu}
                    contextMenuColumn={true}
                    dataTableDefaultView={
                        <DataTableDefaultView
                            icon={FavoriteIcon}
                            messages={['Your favorites list is empty.']}
                            />
                    } />;
            }
        }
    )
);
