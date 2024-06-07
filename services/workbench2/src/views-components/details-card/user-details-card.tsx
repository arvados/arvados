// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Card, CardHeader, WithStyles, withStyles, Typography, Grid } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { UserResource } from 'models/user';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { UserResourceAccountStatus } from 'views-components/data-explorer/renderers';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { Resource } from 'models/resource';
import { ProjectResource } from 'models/project';

type CssRules = 'root' | 'cardHeaderContainer' | 'cardHeader' | 'userNameContainer' | 'accountStatusSection';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        padding: 0,
        minHeight: '3rem',
    },
    userNameContainer: {
        display: 'flex',
        alignItems: 'center',
        minHeight: '2.7rem',
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
        padding: '0.2rem 0.4rem 0.2rem 1rem',
    },
    accountStatusSection: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        paddingLeft: '1rem',
    },
});

const mapStateToProps = ({ auth, selectedResourceUuid, resources, properties }: RootState) => {
    const currentResource = getResource(properties.currentRouteUuid)(resources);
    const frozenByUser = currentResource && getResource((currentResource as ProjectResource).frozenByUuid as string)(resources);
    const frozenByFullName = frozenByUser && (frozenByUser as Resource & { fullName: string }).fullName;
    const isSelected = selectedResourceUuid === properties.currentRouteUuid;

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

type UserCardProps = WithStyles<CssRules> & {
    currentResource: UserResource;
    isAdmin: boolean;
    isSelected: boolean;
    handleCardClick: (resource: any) => void;
};

export const UserCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: UserCardProps) => {
        const { classes, currentResource, handleCardClick, isSelected } = props;
        const { fullName, uuid } = currentResource as UserResource & { fullName: string };

        return (
            <Card
                className={classes.root}
                onClick={() => handleCardClick(uuid)}
                data-cy='user-details-card'
            >
                <Grid
                    container
                    wrap='nowrap'
                    className={classes.cardHeaderContainer}
                >
                    <CardHeader
                        className={classes.cardHeader}
                        title={
                            <section className={classes.userNameContainer}>
                                <Typography
                                    noWrap
                                    variant='h6'
                                >
                                    {fullName}
                                </Typography>
                                <section className={classes.accountStatusSection}>
                                    {!currentResource.isActive && (
                                        <Typography>
                                            <UserResourceAccountStatus uuid={uuid} />
                                        </Typography>
                                    )}
                                </section>
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar />}
                </Grid>
            </Card>
        );
    })
);
