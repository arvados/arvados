// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Card, CardHeader, CardContent, Grid, Switch } from '@material-ui/core';
import { SubprocessFilter } from '~/components/subprocess-filter/subprocess-filter';
import { SubprocessFilterDataProps } from '~/components/subprocess-filter/subprocess-filter';
import { Process } from '~/store/processes/process';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        fontSize: '0.875rem'
    }
});

interface SubprocessesDataProps {
    subprocesses: Array<Process>;
    filters: SubprocessFilterDataProps[];
    onToggle: (filter: SubprocessFilterDataProps) => void;
}

type SubprocessesProps = SubprocessesDataProps & WithStyles<CssRules>;

export const SubprocessesCard = withStyles(styles)(
    ({ classes, filters, subprocesses, onToggle }: SubprocessesProps) => 
        <Card className={classes.root}>
            <CardHeader title="Subprocess and filters" />
            <CardContent>
                <Grid container direction="column" spacing={16}>
                    <Grid item xs={12} container spacing={16}>
                        <SubprocessFilter label='Subprocesses' value={subprocesses.length} />     
                    </Grid>
                    <Grid item xs={12} container spacing={16}>
                        {
                            filters.map(filter => 
                                <SubprocessFilter {...filter} key={filter.key} onToggle={() => onToggle(filter)} />                                                     
                            )
                        }
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);