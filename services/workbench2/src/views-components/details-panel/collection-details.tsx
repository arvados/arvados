// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CollectionIcon, RenameIcon } from 'components/icon/icon';
import { CollectionResource } from 'models/collection';
import { DetailsData } from "./details-data";
import { CollectionDetailsAttributes } from 'views/collection-panel/collection-panel';
import { RootState } from 'store/store';
import { filterResources, getResource, ResourcesState } from 'store/resources/resources';
import { connect } from 'react-redux';
import { Button, Grid, ListItem, StyleRulesCallback, Typography, withStyles, WithStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from 'common/formatters';
import { UserNameFromID } from '../data-explorer/renderers';
import { Dispatch } from 'redux';
import { navigateTo } from 'store/navigation/navigation-action';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { openCollectionUpdateDialog } from 'store/collections/collection-update-actions';
import { resourceIsFrozen } from 'common/frozen-resources';

export type CssRules = 'versionBrowserHeader'
    | 'versionBrowserItem'
    | 'versionBrowserField'
    | 'editButton'
    | 'editIcon'
    | 'tag';

const styles: StyleRulesCallback<CssRules> = theme => ({
    versionBrowserHeader: {
        textAlign: 'center',
        fontWeight: 'bold',
    },
    versionBrowserItem: {
        flexWrap: 'wrap',
    },
    versionBrowserField: {
        textAlign: 'center',
    },
    editIcon: {
        paddingRight: theme.spacing.unit/2,
        fontSize: '1.125rem',
    },
    editButton: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
    tag: {
        marginRight: theme.spacing.unit / 2,
        marginBottom: theme.spacing.unit / 2
    },
});

export class CollectionDetails extends DetailsData<CollectionResource> {

    getIcon(className?: string) {
        return <CollectionIcon className={className} />;
    }

    getTabLabels() {
        return ['Details', 'Versions'];
    }

    getDetails({tabNr}) {
        switch (tabNr) {
            case 0:
                return this.getCollectionInfo();
            case 1:
                return this.getVersionBrowser();
            default:
                return <div />;
        }
    }

    private getCollectionInfo() {
        return <CollectionInfo />;
    }

    private getVersionBrowser() {
        return <CollectionVersionBrowser />;
    }
}

interface CollectionInfoDataProps {
    resources: ResourcesState;
    currentCollection: CollectionResource | undefined;
}

interface CollectionInfoDispatchProps {
    editCollection: (collection: CollectionResource | undefined) => void;
}

const ciMapStateToProps = (state: RootState): CollectionInfoDataProps => {
    return {
        resources: state.resources,
        currentCollection: getResource<CollectionResource>(state.detailsPanel.resourceUuid)(state.resources),
    };
};

const ciMapDispatchToProps = (dispatch: Dispatch): CollectionInfoDispatchProps => ({
    editCollection: (collection: CollectionResource) =>
        dispatch<any>(openCollectionUpdateDialog({
            uuid: collection.uuid,
            name: collection.name,
            description: collection.description,
            properties: collection.properties,
            storageClassesDesired: collection.storageClassesDesired,
        })),
});

type CollectionInfoProps = CollectionInfoDataProps & CollectionInfoDispatchProps & WithStyles<CssRules>;

const CollectionInfo = withStyles(styles)(
    connect(ciMapStateToProps, ciMapDispatchToProps)(
        ({ currentCollection, resources, editCollection, classes }: CollectionInfoProps) =>
            currentCollection !== undefined
                ? <div>
                    <Button
                        disabled={resourceIsFrozen(currentCollection, resources)}
                        className={classes.editButton} variant='contained'
                        data-cy='details-panel-edit-btn' color='primary' size='small'
                        onClick={() => editCollection(currentCollection)}>
                        <RenameIcon className={classes.editIcon} /> Edit
                    </Button>
                    <CollectionDetailsAttributes classes={classes} twoCol={false} item={currentCollection} />
                </div>
                : <div />
    )
);

interface CollectionVersionBrowserProps {
    currentCollection: CollectionResource | undefined;
    versions: CollectionResource[];
}

interface CollectionVersionBrowserDispatchProps {
    showVersion: (c: CollectionResource) => void;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, collection: CollectionResource) => void;
}

const vbMapStateToProps = (state: RootState): CollectionVersionBrowserProps => {
    const currentCollection = getResource<CollectionResource>(state.detailsPanel.resourceUuid)(state.resources);
    const versions = (currentCollection
        && filterResources(rsc =>
            (rsc as CollectionResource).currentVersionUuid === currentCollection.currentVersionUuid)(state.resources)
                .sort((a: CollectionResource, b: CollectionResource) => b.version - a.version) as CollectionResource[])
        || [];
    return { currentCollection, versions };
};

const vbMapDispatchToProps = () =>
    (dispatch: Dispatch): CollectionVersionBrowserDispatchProps => ({
        showVersion: (collection) => dispatch<any>(navigateTo(collection.uuid)),
        handleContextMenu: (event: React.MouseEvent<HTMLElement>, collection: CollectionResource) => {
            const menuKind = dispatch<any>(resourceUuidToContextMenuKind(collection.uuid));
            if (collection && menuKind) {
                dispatch<any>(openContextMenu(event, {
                    name: collection.name,
                    uuid: collection.uuid,
                    description: collection.description,
                    storageClassesDesired: collection.storageClassesDesired,
                    ownerUuid: collection.ownerUuid,
                    isTrashed: collection.isTrashed,
                    kind: collection.kind,
                    menuKind
                }));
            }
        },
    });

const CollectionVersionBrowser = withStyles(styles)(
    connect(vbMapStateToProps, vbMapDispatchToProps)(
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
                            selected={isSelectedVersion}
                            className={classes.versionBrowserItem}>
                            <Grid item xs={2}>
                                <Typography variant="caption" className={classes.versionBrowserField}>
                                    {item.version}
                                </Typography>
                            </Grid>
                            <Grid item xs={4}>
                                <Typography variant="caption" className={classes.versionBrowserField}>
                                    {formatFileSize(item.fileSizeTotal)}
                                </Typography>
                            </Grid>
                            <Grid item xs={6}>
                                <Typography variant="caption" className={classes.versionBrowserField}>
                                    {formatDate(item.modifiedAt)}
                                </Typography>
                            </Grid>
                            <Grid item xs={12}>
                                <Typography variant="caption" className={classes.versionBrowserField}>
                                    Modified by: <UserNameFromID uuid={item.modifiedByUserUuid} />
                                </Typography>
                            </Grid>
                        </ListItem>
                    );
                })}
                </Grid>
            </div>;
        }));
