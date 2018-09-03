// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Typography, Tooltip
} from '@material-ui/core';
import { Process } from '~/store/processes/process';
import { ProcessLogCodeSnippet } from '~/views/process-log-panel/process-log-code-snippet';
import { ProcessLogForm, ProcessLogFormDataProps, ProcessLogFormActionProps } from '~/views/process-log-panel/process-log-form';
import { MoreOptionsIcon, ProcessIcon } from '~/components/icon/icon';
import { ArvadosTheme } from '~/common/custom-theme';
import { CodeSnippetDataProps } from '~/components/code-snippet/code-snippet';

type CssRules = 'card' | 'iconHeader' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        width: '100%'
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700
    },
    link: {
        alignSelf: 'flex-end',
        textAlign: 'right'
    }
});


interface ProcessLogMainCardDataProps {
    process: Process;
}

export type ProcessLogMainCardProps = ProcessLogMainCardDataProps & CodeSnippetDataProps & ProcessLogFormDataProps & ProcessLogFormActionProps;

export const ProcessLogMainCard = withStyles(styles)(
    ({ classes, process, selectedFilter, filters, onChange, lines }: ProcessLogMainCardProps & WithStyles<CssRules>) => 
        <Card className={classes.card}>
            <CardHeader
                avatar={<ProcessIcon className={classes.iconHeader} />}
                action={
                    <div>
                        <IconButton aria-label="More options">
                            <MoreOptionsIcon />
                        </IconButton>
                    </div>
                }
                title={
                    <Tooltip title={process.containerRequest.name}>
                        <Typography noWrap variant="title">
                            {process.containerRequest.name}
                        </Typography>
                    </Tooltip>
                }
                subheader={process.containerRequest.description} />
            <CardContent>
                <Grid container spacing={24} alignItems='center'>
                    <Grid item xs={6}>
                        <ProcessLogForm selectedFilter={selectedFilter} filters={filters} onChange={onChange} />
                    </Grid>
                    <Grid item xs={6} className={classes.link}>
                        <Typography component='div'>
                            Container log for request ardev-xvhdp-q3uqbfxeb6w64pm
                        </Typography>
                    </Grid>
                    <Grid item xs={12}>
                        <ProcessLogCodeSnippet lines={lines}/>
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);