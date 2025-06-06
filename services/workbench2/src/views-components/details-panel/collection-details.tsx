// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CollectionIcon, RenameIcon } from 'components/icon/icon';
import { CollectionResource } from 'models/collection';
import { DetailsData } from "./details-data";
import { RootState } from 'store/store';
import { filterResources, getResource, ResourcesState } from 'store/resources/resources';
import { connect } from 'react-redux';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Button, Grid, ListItem, Typography, Tooltip, Link as ButtonLink } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { formatDate, formatFileSize } from 'common/formatters';
import { UserNameFromID } from '../data-explorer/renderers';
import { Dispatch } from 'redux';
import { navigateTo } from 'store/navigation/navigation-action';
import { openContextMenuAndSelect } from 'store/context-menu/context-menu-actions';
import { openCollectionUpdateDialog } from 'store/collections/collection-update-actions';
import { resourceIsFrozen } from 'common/frozen-resources';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { ResourceWithName, ResponsiblePerson } from 'views-components/data-explorer/renderers';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';

export type CssRules = 'versionBrowserHeader'
    | 'versionBrowserItem'
    | 'versionBrowserField'
    | 'editButton'
    | 'editIcon'
    | 'tag';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
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
        paddingRight: theme.spacing(0.5),
        fontSize: '1.125rem',
    },
    editButton: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5)
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
            const menuKind = dispatch<any>(resourceToMenuKind(collection.uuid));
            if (collection && menuKind) {
                dispatch<any>(openContextMenuAndSelect(event, {
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

interface CollectionDetailsProps {
    item: CollectionResource;
    classes?: any;
    twoCol?: boolean;
    showVersionBrowser?: () => void;
}

export const CollectionDetailsAttributes = (props: CollectionDetailsProps) => {
    const item = props.item;
    const classes = props.classes || { label: '', value: '', button: '', tag: '' };
    const isOldVersion = item && item.currentVersionUuid !== item.uuid;
    const mdSize = props.twoCol ? 6 : 12;
    const showVersionBrowser = props.showVersionBrowser;
    const responsiblePersonRef = React.useRef(null);
    return <Grid container>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label={isOldVersion ? "This version's UUID" : "Collection UUID"}
                linkToUuid={item.uuid} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label={isOldVersion ? "This version's PDH" : "Portable data hash"}
                linkToUuid={item.portableDataHash} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Owner' linkToUuid={item.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
        </Grid>
        <div data-cy="responsible-person-wrapper" ref={responsiblePersonRef}>
            <Grid item xs={12} md={12}>
                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                    label='Responsible person' linkToUuid={item.ownerUuid}
                    uuidEnhancer={(uuid: string) => <ResponsiblePerson uuid={item.uuid} parentRef={responsiblePersonRef.current} />} />
            </Grid>
        </div>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Head version'
                value={isOldVersion ? undefined : 'this one'}
                linkToUuid={isOldVersion ? item.currentVersionUuid : undefined} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute
                classLabel={classes.label} classValue={classes.value}
                label='Version number'
                value={showVersionBrowser !== undefined
                    ? <Tooltip title="Open version browser"><ButtonLink underline='none' className={classes.button} onClick={() => showVersionBrowser()}>
                        {<span data-cy='collection-version-number'>{item.version}</span>}
                    </ButtonLink></Tooltip>
                    : item.version
                }
            />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Created at' value={formatDate(item.createdAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Last modified' value={formatDate(item.modifiedAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Number of files' value={<span data-cy='collection-file-count'>{item.fileCount}</span>} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Content size' value={formatFileSize(item.fileSizeTotal)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Storage classes' value={item.storageClassesDesired ? item.storageClassesDesired.join(', ') : ["default"]} />
        </Grid>

        {/*
            NOTE: The property list should be kept at the bottom, because it spans
            the entire available width, without regards of the twoCol prop.
          */}
        <Grid item xs={12} md={12}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Properties' />
            {item.properties && Object.keys(item.properties).length > 0
                ? Object.keys(item.properties).map(k =>
                    Array.isArray(item.properties[k])
                        ? item.properties[k].map((v: string) =>
                            getPropertyChip(k, v, undefined, classes.tag))
                        : getPropertyChip(k, item.properties[k], undefined, classes.tag))
                : <div className={classes.value}>No properties</div>}
        </Grid>
    </Grid>;
};

