// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Button } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CollectionIcon } from 'components/icon/icon';
import { ArvadosTheme } from 'common/custom-theme';
import { deselectAllOthers, toggleOne } from 'store/multiselect/multiselect-actions';
import { BackIcon } from 'components/icon/icon';
import { COLLECTIONS_CONTENT_ADDRESS_PANEL_ID } from 'store/collections-content-address-panel/collections-content-address-panel-actions';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { Dispatch } from 'redux';
import {
    openContextMenuAndSelect
} from 'store/context-menu/context-menu-actions';
import { ResourceKind } from 'models/resource';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { connect } from 'react-redux';
import { navigateTo } from 'store/navigation/navigation-action';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { createTree } from 'models/tree';
import {
    ResourceName,
    ResourceOwnerName,
    ResourceLastModifiedDate,
    ResourceStatus
} from 'views-components/data-explorer/renderers';
import { getResource, ResourcesState } from 'store/resources/resources';
import { RootState } from 'store/store';
import { CollectionResource } from 'models/collection';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';

type CssRules = 'backLink' | 'backIcon' | 'root' | 'content';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    backLink: {
        fontSize: '12px',
        fontWeight: 600,
        display: 'flex',
        alignItems: 'center',
        padding: theme.spacing(1),
        marginBottom: theme.spacing(1),
        color: theme.palette.grey["500"],
    },
    backIcon: {
        marginRight: theme.spacing(1),
    },
    root: {
        width: '100%',
    },
    content: {
        // reserve space for the content address bar
        height: `calc(100% - ${theme.spacing(7)})`,
    },
});

enum CollectionContentAddressPanelColumnNames {
    COLLECTION_WITH_THIS_ADDRESS = "Collection with this address",
    STATUS = "Status",
    LOCATION = "Location",
    LAST_MODIFIED = "Last modified"
}

export const collectionContentAddressPanelColumns: DataColumns<string, CollectionResource> = [
    {
        name: CollectionContentAddressPanelColumnNames.COLLECTION_WITH_THIS_ADDRESS,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "uuid"},
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: CollectionContentAddressPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceStatus uuid={uuid} />
    },
    {
        name: CollectionContentAddressPanelColumnNames.LOCATION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwnerName uuid={uuid} />
    },
    {
        name: CollectionContentAddressPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.DESC, field: "modifiedAt"},
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

interface CollectionContentAddressPanelActionProps {
    onContextMenu: (resources: ResourcesState) => (event: React.MouseEvent<any>, uuid: string) => void;
    onItemClick: (item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

interface CollectionContentAddressPanelDataProps {
    resources: ResourcesState;
}

const mapStateToProps = ({ resources }: RootState): CollectionContentAddressPanelDataProps => ({
    resources
})

const mapDispatchToProps = (dispatch: Dispatch): CollectionContentAddressPanelActionProps => ({
    onContextMenu: (resources: ResourcesState) => (event, resourceUuid) => {
        const resource = getResource<CollectionResource>(resourceUuid)(resources);
        const menuKind = dispatch<any>(resourceToMenuKind(resourceUuid));
        if (menuKind) {
            dispatch<any>(openContextMenuAndSelect(event, {
                name: resource ? resource.name : '',
                description: resource ? resource.description : '',
                storageClassesDesired: resource ? resource.storageClassesDesired : [],
                uuid: resourceUuid,
                ownerUuid: '',
                kind: ResourceKind.NONE,
                menuKind
            }));
        }
        dispatch<any>(loadDetailsPanel(resourceUuid));
    },
    onItemClick: (uuid: string) => {
        dispatch<any>(toggleOne(uuid))
        dispatch<any>(deselectAllOthers(uuid))
        dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateTo(uuid));
    }
});

interface CollectionContentAddressDataProps {
    match: {
        params: { id: string }
    };
}

export const CollectionsContentAddressPanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<CollectionContentAddressPanelActionProps & CollectionContentAddressPanelDataProps & CollectionContentAddressDataProps & WithStyles<CssRules>> {
            render() {
                return <div className={this.props.classes.root}>
                    <Button
                        onClick={() => window.history.back()}
                        className={this.props.classes.backLink}>
                        <BackIcon className={this.props.classes.backIcon} />
                        Back
                    </Button>
                    <div className={this.props.classes.content}>
                        <DataExplorer
                            id={COLLECTIONS_CONTENT_ADDRESS_PANEL_ID}
                            hideSearchInput
                            onRowClick={this.props.onItemClick}
                            onRowDoubleClick={this.props.onItemDoubleClick}
                            onContextMenu={this.props.onContextMenu(this.props.resources)}
                            contextMenuColumn={false}
                            title={`Content address: ${this.props.match.params.id}`}
                            defaultViewIcon={CollectionIcon}
                            defaultViewMessages={['Collections with this content address not found.']}
                        />
                    </div>
                </div>;
            }
        }
    )
);
