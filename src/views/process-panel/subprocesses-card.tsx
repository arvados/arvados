// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Card, CardHeader, CardContent, Grid, Typography } from '@material-ui/core';
import { SubprocessFilter } from '~/components/subprocess-filter/subprocess-filter';
import { SubprocessFilterDataProps } from '~/components/subprocess-filter/subprocess-filter';

type CssRules = 'root' | 'subtitle' | 'title' | 'gridFilter';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        fontSize: '0.875rem',
        height: '100%'
    },
    subtitle: {
        paddingBottom: '32px!important'
    },
    title: {
        color: theme.customs.colors.grey700
    },
    gridFilter: {
        height: '20px',
        marginBottom: theme.spacing.unit,
        paddingTop: '0px!important',
        paddingBottom: '0px!important',
    }
});

interface SubprocessesDataProps {
    subprocessesAmount: number;
    filters: SubprocessFilterDataProps[];
    onToggle: (status: string) => void;
}

type SubprocessesProps = SubprocessesDataProps & WithStyles<CssRules>;

export const SubprocessesCard = withStyles(styles)(
    ({ classes, filters, subprocessesAmount, onToggle }: SubprocessesProps) =>
        <Card className={classes.root}>
            <CardHeader
                className={classes.title}
                title={
                    <Typography noWrap variant="title" color='inherit'>
                        Subprocess and filters
                </Typography>} />
            <CardContent>
                <Grid container direction="column" spacing={16}>
                    <Grid item xs={12} container spacing={16}>
                        <Grid item md={12} lg={6} className={classes.subtitle}>
                            <SubprocessFilter label='Subprocesses' value={subprocessesAmount} />
                        </Grid>
                        <Grid item md={12} lg={6}/>
                        {
                            filters.map(filter =>
                                <Grid item md={12} lg={6} key={filter.key} spacing={0} className={classes.gridFilter}>
                                    <SubprocessFilter {...filter} onToggle={() => onToggle(filter.label)} />
                                </Grid>
                            )
                        }
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);