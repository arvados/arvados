// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Card, CardHeader, WithStyles, withStyles, Typography, CardContent } from '@material-ui/core';
import { StyleRulesCallback } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1.5rem',
    },
});

const mapStateToProps = (state: RootState) => {
    const currentRoute = state.router.location?.pathname.split('/') || [];
    const currentItemUuid = currentRoute[currentRoute?.length - 1];
    const currentResource = getResource(currentItemUuid)(state.resources);
    return {
        currentResource,
    };
};

type DetailsCardProps = {
    currentResource: any;
};

export const DetailsCard = connect(mapStateToProps)(
    withStyles(styles)((props: DetailsCardProps & WithStyles<CssRules>) => {
        const { classes, currentResource } = props;
        const { name, description, uuid } = currentResource;
        return (
            <Card className={classes.root}>
                {console.log(currentResource)}
                <CardHeader
                    title={
                        <Typography
                            noWrap
                            variant='h6'
                        >
                            {name}
                        </Typography>
                    }
                    subheader={
                        <Typography
                            noWrap
                            variant='body1'
                            color='inherit'
                        >
                            {description ? description.replace(/<[^>]*>/g, '') : '(no-description)'}
                        </Typography>
                    }
                    action={<MultiselectToolbar inputSelectedUuid={uuid} />}
                />
                <CardContent></CardContent>
            </Card>
        );
    })
);
