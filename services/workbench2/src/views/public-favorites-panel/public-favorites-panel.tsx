// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from 'components/data-table/data-column';
import { RouteComponentProps } from 'react-router';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ResourceKind } from 'models/resource';
import { ArvadosTheme } from 'common/custom-theme';
import {
    ProcessStatus,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceType,
    ResourceName,
    ResourceOwnerWithName
} from 'views-components/data-explorer/renderers';
import { PublicFavoriteIcon } from 'components/icon/icon';
import { Dispatch } from 'redux';
import {
    openContextMenuAndSelect,
} from 'store/context-menu/context-menu-actions';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { navigateTo } from 'store/navigation/navigation-action';
import { ContainerRequestState } from "models/container-request";
import { RootState } from 'store/store';
import { createTree } from 'models/tree';
import { getSimpleObjectTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { PUBLIC_FAVORITE_PANEL_ID } from 'store/public-favorites-panel/public-favorites-action';
import { PublicFavoritesState } from 'store/public-favorites/public-favorites-reducer';
import { getResource, ResourcesState } from 'store/resources/resources';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { CollectionResource } from 'models/collection';
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';

type CssRules = "toolbar" | "button" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing(3),
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing(1)
    },
    root: {
        width: '100%',
        boxShadow: "0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)",
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

export const publicFavoritePanelColumns: DataColumns<string, GroupContentsResource> = [
    {
        name: PublicFavoritePanelColumnNames.NAME,
        selected: true,
        configurable: true,
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
        render: uuid => <ResourceOwnerWithName uuid={uuid} />
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
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

interface PublicFavoritePanelDataProps {
    publicFavorites: PublicFavoritesState;
    resources: ResourcesState;
}

interface PublicFavoritePanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (resources: ResourcesState) => (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = ({ publicFavorites, resources }: RootState): PublicFavoritePanelDataProps => ({
    publicFavorites,
    resources,
});

const mapDispatchToProps = (dispatch: Dispatch): PublicFavoritePanelActionProps => ({
    onContextMenu: (resources: ResourcesState) => (event, resourceUuid) => {
        const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
        const kind = dispatch<any>(resourceToMenuKind(resourceUuid));
        if (kind && resource) {
            dispatch<any>(openContextMenuAndSelect(event, {
                name: resource.name,
                description: resource.description,
                storageClassesDesired: (resource as CollectionResource).storageClassesDesired,
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
                dispatch<any>(toggleOne(uuid))
                dispatch<any>(deselectAllOthers(uuid))
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
                return <div className={this.props.classes.root}><DataExplorer
                    id={PUBLIC_FAVORITE_PANEL_ID}
                    onRowClick={this.props.onItemClick}
                    onRowDoubleClick={this.props.onItemDoubleClick}
                    onContextMenu={this.props.onContextMenu(this.props.resources)}
                    contextMenuColumn={false}
                    defaultViewIcon={PublicFavoriteIcon}
                    defaultViewMessages={['Public favorites list is empty.']} />
                </div>;
            }
        }
    )
);
