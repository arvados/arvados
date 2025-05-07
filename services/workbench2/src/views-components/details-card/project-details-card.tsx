// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Card, CardHeader, Typography, Tooltip, Grid } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { ProjectResource } from 'models/project';
import { FavoriteStar, PublicFavoriteStar } from 'views-components/favorite-star/favorite-star';
import { FreezeIcon } from 'components/icon/icon';
import { Resource } from 'models/resource';
import { Dispatch } from 'redux';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { deselectAllOthers } from 'store/multiselect/multiselect-actions';

type CssRules =
    | 'root'
    | 'cardHeaderContainer'
    | 'cardHeader'
    | 'nameSection'
    | 'namePlate'
    | 'faveIcon'
    | 'frozenIcon'
    | 'toolbarStyles';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        padding: 0,
        minHeight: '3rem',
    },
    cardHeaderContainer: {
        width: '100%',
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
    },
    cardHeader: {
        minWidth: '30rem',
        padding: '0.2rem',
    },
    nameSection: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
    },
    namePlate: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        margin: 0,
        minHeight: '2.7rem',
        marginLeft: '.5rem',
    },
    faveIcon: {
        fontSize: '0.8rem',
        margin: 'auto 0 0.2rem 0.3rem',
        color: theme.palette.text.primary,
    },
    frozenIcon: {
        fontSize: '0.5rem',
        marginLeft: '0.3rem',
        height: '1rem',
        color: theme.palette.text.primary,
    },
    toolbarStyles: {
        marginRight: '-0.5rem',
        paddingTop: '4px',
    },
});

const mapStateToProps = ({ auth, selectedResource, resources, properties }: RootState) => {
    const currentResource = getResource(properties.currentRouteUuid)(resources);
    const frozenByUser = currentResource && getResource((currentResource as ProjectResource).frozenByUuid as string)(resources);
    const frozenByFullName = frozenByUser && (frozenByUser as Resource & { fullName: string }).fullName;
    const isSelected = selectedResource.selectedResourceUuid === properties.currentRouteUuid;

    return {
        isAdmin: auth.user?.isAdmin,
        currentResource,
        frozenByFullName,
        isSelected,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleCardClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
        dispatch<any>(setSelectedResourceUuid(uuid));
        dispatch<any>(deselectAllOthers(uuid));
    },
});

type ProjectCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource;
    frozenByFullName: string | undefined;
    isAdmin: boolean;
    isSelected: boolean;
    handleCardClick: (resource: any) => void;
};

export const ProjectCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: ProjectCardProps) => {
        const { classes, currentResource, frozenByFullName, handleCardClick, isSelected } = props;
        const { name, uuid } = currentResource as ProjectResource;

        return (
            <Card
                className={classes.root}
                onClick={() => handleCardClick(uuid)}
                data-cy='project-details-card'
            >
                <Grid
                    container
                    wrap='nowrap'
                    className={classes.cardHeaderContainer}
                >
                    <CardHeader
                        className={classes.cardHeader}
                        title={
                            <section className={classes.nameSection}>
                                <section className={classes.namePlate}>
                                    <Typography
                                        variant='h6'
                                        style={{ marginRight: '1rem' }}
                                    >
                                        {name}
                                    </Typography>

                                    <FavoriteStar
                                        className={classes.faveIcon}
                                        resourceUuid={currentResource.uuid}
                                    />
                                    <PublicFavoriteStar
                                        className={classes.faveIcon}
                                        resourceUuid={currentResource.uuid}
                                    />
                                    {!!frozenByFullName && (
                                        <Tooltip
                                            className={classes.frozenIcon}
                                            disableFocusListener
                                            title={<span>Project was frozen by {frozenByFullName}</span>}
                                        >
                                            <span><FreezeIcon style={{ fontSize: '1.25rem' }} /></span>
                                        </Tooltip>
                                    )}
                                </section>
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar injectedStyles={classes.toolbarStyles} />}
                </Grid>
            </Card>
        );
    })
);
