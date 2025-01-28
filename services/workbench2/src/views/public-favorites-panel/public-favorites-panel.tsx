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
    renderType,
    RenderName,
    RenderOwnerName,
    renderFileSize,
    renderLastModifiedDate,
} from 'views-components/data-explorer/renderers';
import { PublicFavoriteIcon } from 'components/icon/icon';
import { Dispatch } from 'redux';
import {
    openContextMenu,
} from 'store/context-menu/context-menu-actions';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { navigateTo } from 'store/navigation/navigation-action';
import { ContainerRequestState } from "models/container-request";
import { RootState } from 'store/store';
import { createTree } from 'models/tree';
import { getSimpleObjectTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { PUBLIC_FAVORITE_PANEL_ID } from 'store/public-favorites-panel/public-favorites-action';
import { PublicFavoritesState } from 'store/public-favorites/public-favorites-reducer';
import { ResourcesState } from 'store/resources/resources';
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

export const publicFavoritePanelColumns: DataColumns<GroupContentsResource> = [
    {
        name: PublicFavoritePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <RenderName resource={resource} />,
    },
    {
        name: "Status",
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ProcessStatus uuid={resource.uuid} />
    },
    {
        name: PublicFavoritePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getSimpleObjectTypeFilters(),
        render: (resource) => renderType(resource),
    },
    {
        name: PublicFavoritePanelColumnNames.OWNER,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => <RenderOwnerName resource={resource} />
    },
    {
        name: PublicFavoritePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderFileSize(resource),
    },
    {
        name: PublicFavoritePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderLastModifiedDate(resource),
    }
];

interface PublicFavoritePanelDataProps {
    publicFavorites: PublicFavoritesState;
    resources: ResourcesState;
}

interface PublicFavoritePanelActionProps {
    onItemClick: (resource: GroupContentsResource) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, resource: GroupContentsResource) => void;
    onItemDoubleClick: (resource: GroupContentsResource) => void;
}
const mapStateToProps = ({ publicFavorites, resources }: RootState): PublicFavoritePanelDataProps => ({
    publicFavorites,
    resources,
});

const mapDispatchToProps = (dispatch: Dispatch): PublicFavoritePanelActionProps => ({
    onContextMenu: (event, resource: GroupContentsResource) => {
        const kind = dispatch<any>(resourceToMenuKind(resource.uuid));
        if (kind && resource) {
            dispatch<any>(openContextMenu(event, {
                name: resource.name,
                description: resource.description,
                storageClassesDesired: (resource as CollectionResource).storageClassesDesired,
                uuid: resource.uuid,
                ownerUuid: '',
                kind: ResourceKind.NONE,
                menuKind: kind
            }));
        }
        dispatch<any>(loadDetailsPanel(resource.uuid));
    },
    onItemClick: ({uuid}: GroupContentsResource) => {
                dispatch<any>(toggleOne(uuid))
                dispatch<any>(deselectAllOthers(uuid))
                dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: ({uuid}: GroupContentsResource) => {
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
                    onContextMenu={this.props.onContextMenu}
                    contextMenuColumn={false}
                    defaultViewIcon={PublicFavoriteIcon}
                    defaultViewMessages={['Public favorites list is empty.']} />
                </div>;
            }
        }
    )
);
