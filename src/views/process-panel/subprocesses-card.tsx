// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Card, CardHeader, CardContent, Grid, Switch, Typography } from '@material-ui/core';
import { SubprocessFilter } from '~/components/subprocess-filter/subprocess-filter';
import { SubprocessFilterDataProps } from '~/components/subprocess-filter/subprocess-filter';
import { Process } from '~/store/processes/process';

type CssRules = 'root' | 'subtitle' | 'title';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        fontSize: '0.875rem'
    },
    subtitle: {
        paddingBottom: '28px!important'
    },
    title: {
        color: theme.customs.colors.grey700
    }
});

interface SubprocessesDataProps {
    subprocesses: Array<Process>;
    filters: SubprocessFilterDataProps[];
    onToggle: (status: string) => void;
}

type SubprocessesProps = SubprocessesDataProps & WithStyles<CssRules>;

export const SubprocessesCard = withStyles(styles)(
    ({ classes, filters, subprocesses, onToggle }: SubprocessesProps) =>
        <Card className={classes.root}>
            <CardHeader 
                className={classes.title}
                title={
                    <Typography noWrap variant="title" color='inherit'>
                        Subprocess and filters
                </Typography>} />
            <CardContent>
                <Grid container direction="column" spacing={16}>
                    <Grid item xs={12} container spacing={16} className={classes.subtitle}>
                        <SubprocessFilter label='Subprocesses' value={subprocesses.length} />
                    </Grid>
                    <Grid item xs={12} container spacing={16}>
                        {
                            filters.map(filter =>
                                <SubprocessFilter {...filter} key={filter.key} onToggle={() => onToggle(filter.label)} />
                            )
                        }
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);