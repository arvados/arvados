// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Card, CardHeader, Typography, Grid, Tooltip } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { CollectionResource } from 'models/collection';
import { IllegalNamingWarning } from 'components/warning/warning';
import { GroupResource } from 'models/group';
import { UserResource } from 'models/user';
import { resourceIsFrozen } from 'common/frozen-resources';
import { ReadOnlyIcon } from 'components/icon/icon';

type CssRules = 'root' | 'cardHeaderContainer' | 'cardHeader' | 'readOnlyIcon' | 'nameContainer';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        padding: 0,
        minHeight: '3rem',
    },
    nameContainer: {
        display: 'flex',
        alignItems: 'center',
        minHeight: '2.7rem',
    },
    cardHeaderContainer: {
        width: '100%',
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
    },
    cardHeader: {
        minWidth: '30rem',
        padding: '0.2rem 0.4rem 0.2rem 1rem',
    },
    readOnlyIcon: {
        marginLeft: theme.spacing(1),
        fontSize: 'small',
    },
});

const mapStateToProps = ({ auth, selectedResource, resources, properties }: RootState): Pick<CollectionCardProps, 'currentUserUUID' | 'currentResource' | 'isSelected' | 'itemOwner' | 'isFrozen'> => {
    const currentResource = getResource<CollectionResource>(properties.currentRouteUuid)(resources);
    const isSelected = selectedResource.selectedResourceUuid === properties.currentRouteUuid;
    const itemOwner = currentResource ? getResource<GroupResource | UserResource>(currentResource.ownerUuid)(resources) : undefined;
    const isFrozen = (currentResource && resourceIsFrozen(currentResource, resources)) || false;

    return {
        currentUserUUID: auth.user?.uuid || '',
        currentResource,
        isSelected,
        itemOwner,
        isFrozen,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleCardClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
        dispatch<any>(setSelectedResourceUuid(uuid));
        dispatch<any>(deselectAllOthers(uuid));
    },
});

type CollectionCardProps = WithStyles<CssRules> & {
    currentResource: CollectionResource | undefined;
    isSelected: boolean;
    currentUserUUID: string;
    itemOwner: GroupResource | UserResource | undefined;
    isFrozen: boolean;
    handleCardClick: (resource: any) => void;
};

export const CollectionCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: CollectionCardProps) => {
        const { classes, currentResource, handleCardClick, isSelected, currentUserUUID, itemOwner, isFrozen } = props;
        if (!currentResource) return null;
        const { name, uuid } = currentResource;

        const [isWritable, setIsWritable] = useState(false);

        useEffect(() => {
            const isWritable = checkIsWritable(currentResource, itemOwner, currentUserUUID, isFrozen);
            setIsWritable(isWritable);
            // eslint-disable-next-line react-hooks/exhaustive-deps
        }, [currentResource, currentUserUUID, isFrozen]);

        return (
            <Card
                className={classes.root}
                onClick={() => handleCardClick(uuid)}
                data-cy='collection-details-card'
            >
                <Grid
                    container
                    wrap='nowrap'
                    className={classes.cardHeaderContainer}
                >
                    <CardHeader
                        className={classes.cardHeader}
                        title={
                            <section className={classes.nameContainer}>
                                <IllegalNamingWarning name={name} />
                                <Typography
                                    variant='h6'
                                >
                                    {name}
                                </Typography>
                                {!isWritable &&
                                    <Tooltip title="Read-only">
                                        <span><ReadOnlyIcon data-cy="read-only-icon" className={classes.readOnlyIcon} /></span>
                                    </Tooltip>
                                }
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar />}
                </Grid>
            </Card>
        );
    })
);

const checkIsWritable = (item: CollectionResource, itemOwner: GroupResource | UserResource | undefined, currentUserUUID: string, isFrozen: boolean): boolean => {
    const isCurrentVersion = item.currentVersionUuid === item.uuid

    let isWritable = false;

    if (isCurrentVersion) {
        if (item.ownerUuid === currentUserUUID) {
            isWritable = true;
        } else {
            if (itemOwner) {
                isWritable = itemOwner.canWrite;
            }
        }
    }
    if (isWritable) {
        isWritable = !isFrozen;
    }
    return isWritable;
}
