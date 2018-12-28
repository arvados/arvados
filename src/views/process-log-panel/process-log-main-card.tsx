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
import { DefaultView } from '~/components/default-view/default-view';

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

export interface ProcessLogMainCardActionProps {
    onContextMenu: (event: React.MouseEvent<any>, process: Process) => void;
}

export type ProcessLogMainCardProps = ProcessLogMainCardDataProps
    & ProcessLogMainCardActionProps
    & CodeSnippetDataProps
    & ProcessLogFormDataProps
    & ProcessLogFormActionProps;

export const ProcessLogMainCard = withStyles(styles)(
    ({ classes, process, selectedFilter, filters, onChange, lines, onContextMenu }: ProcessLogMainCardProps & WithStyles<CssRules>) =>
        <Grid item xs={12}>
            <Link to={`/processes/${process.containerRequest.uuid}`} className={classes.backLink}>
                <BackIcon className={classes.backIcon} /> Back
            </Link>
            <Card className={classes.card}>
                <CardHeader
                    avatar={<ProcessIcon className={classes.iconHeader} />}
                    action={
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton onClick={event => onContextMenu(event, process)} aria-label="More options">
                                <MoreOptionsIcon />
                            </IconButton>
                        </Tooltip>}
                    title={
                        <Tooltip title={process.containerRequest.name} placement="bottom-start">
                            <Typography noWrap variant='h6' className={classes.title}>
                                {process.containerRequest.name}
                            </Typography>
                        </Tooltip>}
                    subheader={process.containerRequest.description} />
                <CardContent>
                    {lines.length > 0
                        ? < Grid
                            container
                            spacing={24}
                            direction='column'>
                            <Grid container item>
                                <Grid item xs={6}>
                                    <ProcessLogForm selectedFilter={selectedFilter} filters={filters} onChange={onChange} />
                                </Grid>
                                <Grid item xs={6} className={classes.link}>
                                    <Typography component='div'>
                                        Go to Log collection
                                </Typography>
                                </Grid>
                            </Grid>
                            <Grid item xs>
                                <ProcessLogCodeSnippet lines={lines} />
                            </Grid>
                        </Grid>
                        : <DefaultView
                            icon={ProcessIcon}
                            messages={['No logs yet']} />
                    }
                </CardContent>
            </Card>
        </Grid >
);