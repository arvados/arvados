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
import {
    openContextMenu,
    resourceUuidToContextMenuKind
} from '~/store/context-menu/context-menu-actions';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { navigateTo } from '~/store/navigation/navigation-action';
import { ContainerRequestState } from "~/models/container-request";
import { FavoritesState } from '~/store/favorites/favorites-reducer';
import { RootState } from '~/store/store';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { createTree } from '~/models/tree';
import { getSimpleObjectTypeFilters } from '~/store/resource-type-filters/resource-type-filters';
import { getResource, ResourcesState } from '~/store/resources/resources';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { GroupClass, GroupResource } from '~/models/group';
import { getProperty } from '~/store/properties/properties';
import { PROJECT_PANEL_CURRENT_UUID } from '~/store/project-panel/project-panel-action';

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
        name: FavoritePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getSimpleObjectTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />
    },
    {
        name: FavoritePanelColumnNames.OWNER,
        selected: false,
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
        sortDirection: SortDirection.DESC,
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

interface FavoritePanelDataProps {
    currentItemId: any;
    favorites: FavoritesState;
    resources: ResourcesState;
    userUuid: string;
}

interface FavoritePanelActionProps {
    onItemClick: (item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = (state : RootState): FavoritePanelDataProps => ({
    favorites: state.favorites,
    resources: state.resources,
    userUuid: state.auth.user!.uuid,
    currentItemId: getProperty(PROJECT_PANEL_CURRENT_UUID)(state.properties),
});

type FavoritePanelProps = FavoritePanelDataProps & FavoritePanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const FavoritePanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<FavoritePanelProps> {

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const { resources } = this.props;
                const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
                
                let readonly = false;
                const project = getResource<GroupResource>(this.props.currentItemId)(resources);
                
                if (project && project.groupClass === GroupClass.FILTER) {
                    readonly = true;
                }

                const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(resourceUuid, readonly));

                if (menuKind&& resource) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: resource.name,
                        uuid: resource.uuid,
                        ownerUuid: resource.ownerUuid,
                        isTrashed: ('isTrashed' in resource) ? resource.isTrashed: false,
                        kind: resource.kind,
                        menuKind,
                        description: resource.description,
                    }));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            }

            render() {
                return <DataExplorer
                    id={FAVORITE_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
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
