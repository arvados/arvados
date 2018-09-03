// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Link } from 'react-router-dom';
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
import { BackIcon } from '~/components/icon/icon';

type CssRules = 'backLink' | 'backIcon' | 'card' | 'title' | 'iconHeader' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    backLink: {
        fontSize: '1rem',
        fontWeight: 600,
        display: 'flex',
        alignItems: 'center',
        textDecoration: 'none',
        padding: theme.spacing.unit,
        color: theme.palette.grey["700"],
    },
    backIcon: {
        marginRight: theme.spacing.unit
    },
    card: {
        width: '100%'
    },
    title: {
        color: theme.palette.grey["700"]
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
        <Grid item xs={12}>
            <Link to={`/processes/${process.containerRequest.uuid}`} className={classes.backLink}>
                <BackIcon className={classes.backIcon}/> Back
            </Link>
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
                        <Tooltip title={process.containerRequest.name} placement="bottom-start">
                            <Typography noWrap variant="title" className={classes.title}>
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
                                Go to Log collection
                            </Typography>
                        </Grid>
                        <Grid item xs={12}>
                            <ProcessLogCodeSnippet lines={lines}/>
                        </Grid>
                    </Grid>
                </CardContent>
            </Card>
        </Grid>
);