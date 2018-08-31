// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Chip, Typography, Tooltip
} from '@material-ui/core';
import * as classnames from "classnames";
import { ArvadosTheme } from '~/common/custom-theme';
import { MoreOptionsIcon, ProcessIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { Process } from '~/store/processes/process';
import { getProcessStatus } from '~/store/processes/process';
import { getStatusColor } from '~/views/process-panel/process-panel-root';

type CssRules = 'card' | 'iconHeader' | 'label' | 'value' | 'chip' | 'link' | 'content' | 'title' | 'avatar'
    | 'headerActive' | 'headerCompleted' | 'headerQueued' | 'headerFailed' | 'headerCanceled';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: theme.spacing.unit * 2
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing.unit * 0.5
    },
    label: {
        display: 'flex',
        justifyContent: 'flex-end',
        fontSize: '0.875rem',
        marginRight: theme.spacing.unit * 3,
        paddingRight: theme.spacing.unit
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem',
    },
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    chip: {
        height: theme.spacing.unit * 3,
        width: theme.spacing.unit * 12,
        color: theme.palette.common.white,
        fontSize: '0.875rem',
        borderRadius: theme.spacing.unit * 0.625,
    },
    content: {
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 2,
            paddingTop: '0px'
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5
    },
    headerActive: {
        backgroundColor: theme.customs.colors.blue500,
    },
    headerCompleted: {
        backgroundColor: theme.customs.colors.green700,
    },
    headerQueued: {
        backgroundColor: theme.customs.colors.grey500,
    },
    headerFailed: {
        backgroundColor: theme.customs.colors.red900,
    },
    headerCanceled: {
        backgroundColor: theme.customs.colors.red900,
    },
});

export interface ProcessInformationCardDataProps {
    process: Process;
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

type ProcessInformationCardProps = ProcessInformationCardDataProps & WithStyles<CssRules>;

export const ProcessInformationCard = withStyles(styles)(
    ({ classes, process, onContextMenu }: ProcessInformationCardProps) =>
        <Card className={classes.card}>
            <CardHeader
                classes={{
                    content: classes.title,
                    avatar: classes.avatar
                }}
                avatar={<ProcessIcon className={classes.iconHeader} />}
                action={
                    <div>
                        <Chip label={getProcessStatus(process)}
                            className={classnames([classes.chip, getStatusColor(getProcessStatus(process), classes)])} />
                        <IconButton
                            aria-label="More options"
                            onClick={event => onContextMenu(event)}>
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
            <CardContent className={classes.content}>
                <Grid container>
                    <Grid item xs={6}>
                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                            label='From' value={process.container ? process.container.startedAt : 'N/A'} />
                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                            label='To' value={process.container ? process.container.finishedAt : 'N/A'} />
                        <DetailsAttribute classLabel={classes.label} classValue={classes.link}
                            label='Workflow' value='???' />
                    </Grid>
                    <Grid item xs={6}>
                        <DetailsAttribute classLabel={classes.link} label='Outputs' />
                        <DetailsAttribute classLabel={classes.link} label='Inputs' />
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);