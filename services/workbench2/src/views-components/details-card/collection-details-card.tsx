// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { useEffect, useState } from 'react';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { Card, CardHeader, Typography, CardContent, Tooltip, Collapse, Grid, IconButton } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Link } from 'react-router-dom';
import { ArvadosTheme } from 'common/custom-theme';
import { WithStyles, withStyles } from '@mui/styles';
import { ReadOnlyIcon, CollectionIcon, CollectionOldVersionIcon } from 'components/icon/icon';
import { RootState } from 'store/store';
import { getResource, ResourcesState } from 'store/resources/resources';
import { CollectionResource } from 'models/collection';
import { IllegalNamingWarning } from 'components/warning/warning';
import { CollectionDetailsAttributes } from 'views/collection-panel/collection-panel';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { navigateToProcess } from 'store/collection-panel/collection-panel-action';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { getCollectionUrl } from 'models/collection';
import { UserResource } from 'models/user';
import { GroupResource } from 'models/group';

type CssRules = 'root' | 'mainGrid';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    mainGrid: {
        display: 'flex',
        flexDirection: 'column',
        marginTop: '.5rem',
        paddingTop: '0px',
        paddingBottom: '0px',
        paddingLeft: '.5rem',
        paddingRight: '.5rem',
    },
});

type CollectionDetailsCardDataProps = {
    currentRouteUuid: string | undefined;
    currentUserUUID: string | undefined;
    selectedResourceUuid: string | undefined;
    resources: ResourcesState;
};

type CollectionDetailsCardActionProps = {
    navigateToProcess: (uuid: string) => void;
    openDetailsPanel: (uuid: string) => void;
    setSelectedResourceUuid: (uuid: string) => void;
};

type CollectionDetailsCardProps = CollectionDetailsCardDataProps & CollectionDetailsCardActionProps;

const mapStateToProps = (state: RootState): CollectionDetailsCardDataProps => {
    return {
        currentRouteUuid: state.properties.currentRouteUuid,
        currentUserUUID: state.auth.user?.uuid,
        selectedResourceUuid: state.selectedResource.selectedResourceUuid,
        resources: state.resources,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): CollectionDetailsCardActionProps => {
    return {
        navigateToProcess: (uuid: string) => dispatch<any>(navigateToProcess(uuid)),
        openDetailsPanel: (uuid: string) => dispatch<any>(openDetailsPanel(uuid)),
        setSelectedResourceUuid: (uuid: string) => dispatch<any>(setSelectedResourceUuid(uuid)),
    };
};

export const CollectionDetailsCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)(
        ({
            currentRouteUuid,
            currentUserUUID,
            resources,
            classes,
            selectedResourceUuid,
            setSelectedResourceUuid,
            openDetailsPanel,
        }: CollectionDetailsCardProps & WithStyles<CssRules>) => {
            const [collection, setCollection] = useState<CollectionResource | null>(null);
            const [showDescription, setShowDescription] = useState(false);
            const [showDetails, setShowDetails] = useState(false);
            const [isCurrentVersion, setIsCurrentVersion] = useState(false);
            const [isWritable, setIsWritable] = useState(false);
            const [isSelected, setIsSelected] = useState(false);

            const hasDescription = !!(collection?.description && collection?.description.length > 0);

            useEffect(() => {
                if (currentRouteUuid) setAndSelectCollection();
            }, [currentRouteUuid]);

            useEffect(() => {
                if (collection) {
                    setIsCurrentVersion(collection.uuid === collection.currentVersionUuid);
                }
            }, [collection]);

            useEffect(() => {
                setIsSelected(currentRouteUuid === selectedResourceUuid);
                setIsWritable(checkIsWritable(collection, currentUserUUID));
            }, [currentRouteUuid, selectedResourceUuid]);

            const setAndSelectCollection = () => {
                const fetchedCollection = getResource<CollectionResource>(currentRouteUuid)(resources);
                if (fetchedCollection) {
                    setCollection(fetchedCollection);
                    setSelectedResourceUuid(fetchedCollection.uuid);
                }
            };

            const checkIsWritable = (item: CollectionResource | null, currentUserUUID: string | undefined): boolean => {
                const itemOwner = collection ? getResource<GroupResource | UserResource>(collection.ownerUuid)(resources) : undefined;
                let isWritable = false;
                if (item && isCurrentVersion) {
                    if (item.ownerUuid === currentUserUUID) {
                        isWritable = true;
                    } else {
                        if (itemOwner) {
                            isWritable = itemOwner.canWrite;
                        }
                    }
                }
                return isWritable;
            };

            return collection ? (
                <Card className={classes.root}>
                    <Grid
                        container
                        wrap='nowrap'
                        className={classes.mainGrid}
                    >
                        <CardHeader
                            avatar={
                                <IconButton
                                    onClick={() => openDetailsPanel(collection.uuid)}
                                    size='large'
                                >
                                    {isCurrentVersion ? <CollectionIcon /> : <CollectionOldVersionIcon />}
                                </IconButton>
                            }
                            title={
                                <span>
                                    <IllegalNamingWarning name={collection.name} />
                                    {collection.name}
                                    {!isWritable || (
                                        <Tooltip title='Read-only'>
                                            <span>
                                                <ReadOnlyIcon data-cy='read-only-icon' />
                                            </span>
                                        </Tooltip>
                                    )}
                                    {hasDescription && (
                                        <Collapse
                                            in={showDescription}
                                            collapsedSize={'1rem'}
                                        >
                                            <section
                                                data-cy='collection-description'
                                                onClick={() => setShowDescription(!showDescription)}
                                            >
                                                <Typography
                                                    component='div'
                                                    //dangerouslySetInnerHTML is ok here only if description is sanitized,
                                                    //which it is before it is loaded into the redux store
                                                    dangerouslySetInnerHTML={{ __html: collection.description }}
                                                />
                                            </section>
                                        </Collapse>
                                    )}
                                </span>
                            }
                        />
                        {isSelected && <MultiselectToolbar />}
                        <CardContent>
                            <Collapse
                                in={showDetails}
                                collapsedSize={'1rem'}
                            >
                                <section
                                    data-cy='collection-details'
                                    onClick={() => setShowDetails(!showDetails)}
                                >
                                    <CollectionDetailsAttributes
                                        item={collection}
                                        twoCol={true}
                                        showVersionBrowser={() => openDetailsPanel(collection.uuid)}
                                    />
                                    {(collection.properties.container_request || collection.properties.containerRequest) && (
                                        <span onClick={() => navigateToProcess(collection.properties.container_request || collection.properties.containerRequest)}>
                                            <DetailsAttribute label='Link to process' />
                                        </span>
                                    )}
                                </section>
                            </Collapse>
                            {!isCurrentVersion && (
                                <Typography variant='caption'>
                                    This is an old version. Make a copy to make changes. Go to the <Link to={getCollectionUrl(collection.currentVersionUuid)}>head version</Link>{' '}
                                    for sharing options.
                                </Typography>
                            )}
                        </CardContent>
                    </Grid>
                </Card>
            ) : (
                <div>No collection</div>
            );
        }
    )
);
