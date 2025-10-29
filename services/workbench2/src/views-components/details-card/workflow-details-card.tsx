// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Card, CardHeader, Typography, Grid } from '@mui/material';
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
import { WorkflowResource } from 'models/workflow';

type CssRules = 'root' | 'cardHeaderContainer' | 'cardHeader' | 'nameContainer';

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
});

const mapStateToProps = ({ auth, selectedResource, resources, properties }: RootState) => {
    const currentResource = getResource(properties.currentRouteUuid)(resources);
    const isSelected = selectedResource.selectedResourceUuid === properties.currentRouteUuid;

    return {
        isAdmin: auth.user?.isAdmin,
        currentResource,
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

type WorkflowCardProps = WithStyles<CssRules> & {
    currentResource: WorkflowResource;
    isSelected: boolean;
    handleCardClick: (resource: any) => void;
};

export const WorkflowCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: WorkflowCardProps) => {
        const { classes, currentResource, handleCardClick, isSelected } = props;
        const { name, uuid } = currentResource;

        return (
            <Card
                className={classes.root}
                onClick={() => handleCardClick(uuid)}
                data-cy='workflow-details-card'
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
                                <Typography
                                    variant='h6'
                                >
                                    {name}
                                </Typography>
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar />}
                </Grid>
            </Card>
        );
    })
);
