// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Chip, Typography, Tooltip, Button
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon, MoreOptionsIcon, ProcessIcon } from 'components/icon/icon';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { Process } from 'store/processes/process';
import { getProcessStatus, getProcessStatusColor } from 'store/processes/process';
import { formatDate } from 'common/formatters';
import classNames from 'classnames';
import { ContainerState } from 'models/container';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';

type CssRules = 'card' | 'iconHeader' | 'label' | 'value' | 'chip' | 'link' | 'content' | 'title' | 'avatar' | 'cancelButton';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
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
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5
    },
    cancelButton: {
        paddingRight: theme.spacing.unit * 2,
        fontSize: '14px',
        color: theme.customs.colors.red900,
        "&:hover": {
            cursor: 'pointer'
        }
    }
});

export interface ProcessInformationCardDataProps {
    process: Process;
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
    openProcessInputDialog: (uuid: string) => void;
    navigateToOutput: (uuid: string) => void;
    openWorkflow: (uuid: string) => void;
    cancelProcess: (uuid: string) => void;
}

type ProcessInformationCardProps = ProcessInformationCardDataProps & WithStyles<CssRules, true> & MPVPanelProps;

export const ProcessInformationCard = withStyles(styles, { withTheme: true })(
    ({ classes, process, onContextMenu, theme, openProcessInputDialog, navigateToOutput, openWorkflow, cancelProcess, doHidePanel, panelName }: ProcessInformationCardProps) => {
        const { container } = process;
        const startedAt = container ? formatDate(container.startedAt) : 'N/A';
        const finishedAt = container ? formatDate(container.finishedAt) : 'N/A';
        return <Card className={classes.card}>
            <CardHeader
                classes={{
                    content: classes.title,
                    avatar: classes.avatar
                }}
                avatar={<ProcessIcon className={classes.iconHeader} />}
                action={
                    <div>
                        {process.container && process.container.state === ContainerState.RUNNING &&
                            <span className={classes.cancelButton} onClick={() => cancelProcess(process.containerRequest.uuid)}>Cancel</span>}
                        <Chip label={getProcessStatus(process)}
                            className={classes.chip}
                            style={{ backgroundColor: getProcessStatusColor(getProcessStatus(process), theme as ArvadosTheme) }} />
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton
                                aria-label="More options"
                                onClick={event => onContextMenu(event)}>
                                <MoreOptionsIcon />
                            </IconButton>
                        </Tooltip>
                        { doHidePanel &&
                        <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                            <Button onClick={doHidePanel}><CloseIcon /></Button>
                        </Tooltip> }
                    </div>
                }
                title={
                    <Tooltip title={process.containerRequest.name} placement="bottom-start">
                        <Typography noWrap variant='h6' color='inherit'>
                            {process.containerRequest.name}
                        </Typography>
                    </Tooltip>
                }
                subheader={
                    <Tooltip title={getDescription(process)} placement="bottom-start">
                        <Typography noWrap variant='body1' color='inherit'>
                            {getDescription(process)}
                        </Typography>
                    </Tooltip>} />
            <CardContent className={classes.content}>
                <Grid container>
                    <Grid item xs={6}>
                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                            label='From'
                            value={startedAt} />
                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                            label='To'
                            value={finishedAt} />
                        {process.containerRequest.properties.workflowUuid &&
                            <span onClick={() => openWorkflow(process.containerRequest.properties.workflowUuid)}>
                                <DetailsAttribute classLabel={classes.label} classValue={classNames(classes.value, classes.link)}
                                    label='Workflow' value={process.containerRequest.properties.workflowName} />
                            </span>}
                    </Grid>
                    <Grid item xs={6}>
                        <span onClick={() => navigateToOutput(process.containerRequest.outputUuid!)}>
                            <DetailsAttribute classLabel={classes.link} label='Outputs' />
                        </span>
                        <span onClick={() => openProcessInputDialog(process.containerRequest.uuid)}>
                            <DetailsAttribute classLabel={classes.link} label='Inputs' />
                        </span>
                    </Grid>
                </Grid>
            </CardContent>
        </Card>;
    }
);

const getDescription = (process: Process) =>
    process.containerRequest.description || '(no-description)';
