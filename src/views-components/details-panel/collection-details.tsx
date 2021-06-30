// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CollectionIcon } from 'components/icon/icon';
import { CollectionResource } from 'models/collection';
import { DetailsData } from "./details-data";
import { CollectionDetailsAttributes } from 'views/collection-panel/collection-panel';
import { RootState } from 'store/store';
import { filterResources, getResource } from 'store/resources/resources';
import { connect } from 'react-redux';
import { Grid, ListItem, StyleRulesCallback, Typography, withStyles, WithStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from 'common/formatters';
import { Dispatch } from 'redux';
import { navigateTo } from 'store/navigation/navigation-action';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';

export type CssRules = 'versionBrowserHeader' | 'versionBrowserItem';

const styles: StyleRulesCallback<CssRules> = theme => ({
    versionBrowserHeader: {
        textAlign: 'center',
        fontWeight: 'bold',
    },
    versionBrowserItem: {
        textAlign: 'center',
    }
});

export class CollectionDetails extends DetailsData<CollectionResource> {

    getIcon(className?: string) {
        return <CollectionIcon className={className} />;
    }

    getTabLabels() {
        return ['Details', 'Versions'];
    }

    getDetails(tabNumber: number) {
        switch (tabNumber) {
            case 0:
                return this.getCollectionInfo();
            case 1:
                return this.getVersionBrowser();
            default:
                return <div />;
        }
    }

    private getCollectionInfo() {
        return <CollectionDetailsAttributes twoCol={false} item={this.item} />;
    }

    private getVersionBrowser() {
        return <CollectionVersionBrowser />;
    }
}

interface CollectionVersionBrowserProps {
    currentCollection: CollectionResource | undefined;
    versions: CollectionResource[];
}

interface CollectionVersionBrowserDispatchProps {
    showVersion: (c: CollectionResource) => void;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, collection: CollectionResource) => void;
}

const mapStateToProps = (state: RootState): CollectionVersionBrowserProps => {
    const currentCollection = getResource<CollectionResource>(state.detailsPanel.resourceUuid)(state.resources);
    const versions = currentCollection
        && filterResources(rsc =>
            (rsc as CollectionResource).currentVersionUuid === currentCollection.currentVersionUuid)(state.resources)
                .sort((a: CollectionResource, b: CollectionResource) => b.version - a.version) as CollectionResource[]
        || [];
    return { currentCollection, versions };
};

const mapDispatchToProps = () =>
    (dispatch: Dispatch): CollectionVersionBrowserDispatchProps => ({
        showVersion: (collection) => dispatch<any>(navigateTo(collection.uuid)),
        handleContextMenu: (event: React.MouseEvent<HTMLElement>, collection: CollectionResource) => {
            const menuKind = dispatch<any>(resourceUuidToContextMenuKind(collection.uuid));
            if (collection && menuKind) {
                dispatch<any>(openContextMenu(event, {
                    name: collection.name,
                    uuid: collection.uuid,
                    ownerUuid: collection.ownerUuid,
                    isTrashed: collection.isTrashed,
                    kind: collection.kind,
                    menuKind
                }));
            }
        },
    });

const CollectionVersionBrowser = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ currentCollection, versions, showVersion, handleContextMenu, classes }: CollectionVersionBrowserProps & CollectionVersionBrowserDispatchProps & WithStyles<CssRules>) => {
            return <div data-cy="collection-version-browser">
                <Grid container>
                    <Grid item xs={2}>
                        <Typography variant="caption" className={classes.versionBrowserHeader}>
                            Nr
                        </Typography>
                    </Grid>
                    <Grid item xs={4}>
                        <Typography variant="caption" className={classes.versionBrowserHeader}>
                            Size
                        </Typography>
                    </Grid>
                    <Grid item xs={6}>
                        <Typography variant="caption" className={classes.versionBrowserHeader}>
                            Date
                        </Typography>
                    </Grid>
                { versions.map(item => {
                    const isSelectedVersion = !!(currentCollection && currentCollection.uuid === item.uuid);
                    return (
                        <ListItem button style={{padding: '4px'}}
                            data-cy={`collection-version-browser-select-${item.version}`}
                            key={item.version}
                            onClick={e => showVersion(item)}
                            onContextMenu={event => handleContextMenu(event, item)}
                            selected={isSelectedVersion}>
                            <Grid item xs={2}>
                                <Typography variant="caption" className={classes.versionBrowserItem}>
                                    {item.version}
                                </Typography>
                            </Grid>
                            <Grid item xs={4}>
                                <Typography variant="caption" className={classes.versionBrowserItem}>
                                    {formatFileSize(item.fileSizeTotal)}
                                </Typography>
                            </Grid>
                            <Grid item xs={6}>
                                <Typography variant="caption" className={classes.versionBrowserItem}>
                                    {formatDate(item.modifiedAt)}
                                </Typography>
                            </Grid>
                        </ListItem>
                    );
                })}
                </Grid>
            </div>;
        }));