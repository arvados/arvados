// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { CollectionIcon } from '~/components/icon/icon';
import { CollectionResource } from '~/models/collection';
import { DetailsData } from "./details-data";
import { CollectionDetailsAttributes } from '~/views/collection-panel/collection-panel';
import { RootState } from '~/store/store';
import { filterResources, getResource } from '~/store/resources/resources';
import { connect } from 'react-redux';
import { Grid, ListItem, StyleRulesCallback, Typography, withStyles, WithStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from '~/common/formatters';
import { Dispatch } from 'redux';
import { navigateTo } from '~/store/navigation/navigation-action';

export type CssRules = 'versionBrowserHeader' | 'selectedVersion';

const styles: StyleRulesCallback<CssRules> = theme => ({
    versionBrowserHeader: {
        textAlign: 'center',
        fontWeight: 'bold'
    },
    selectedVersion: {
        fontWeight: 'bold'
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
    });

const CollectionVersionBrowser = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ currentCollection, versions, showVersion, classes }: CollectionVersionBrowserProps & CollectionVersionBrowserDispatchProps & WithStyles<CssRules>) => {
            return <>
                <Grid container justify="space-between">
                    <Typography variant="caption" className={classes.versionBrowserHeader}>
                        Version
                    </Typography>
                    <Typography variant="caption" className={classes.versionBrowserHeader}>
                        Size
                    </Typography>
                    <Typography variant="caption" className={classes.versionBrowserHeader}>
                        Date
                    </Typography>
                </Grid>
                { versions.map(item => {
                    const isSelectedVersion = !!(currentCollection && currentCollection.uuid === item.uuid);
                    return (
                        <ListItem button
                            className={isSelectedVersion ? 'selectedVersion' : ''}
                            key={item.version}
                            onClick={e => showVersion(item)}
                            selected={isSelectedVersion}>
                            <Grid container justify="space-between">
                                <Typography variant="caption">
                                    {item.version}
                                </Typography>
                                <Typography variant="caption">
                                    {formatFileSize(item.fileSizeTotal)}
                                </Typography>
                                <Typography variant="caption">
                                    {formatDate(item.modifiedAt)}
                                </Typography>
                            </Grid>
                        </ListItem>
                    );
                })}
            </>;
        }));