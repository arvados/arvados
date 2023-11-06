// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardHeader,
    IconButton,
    CardContent,
    Tooltip,
    Typography,
    Button,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon, MoreVerticalIcon, ProcessIcon, StartIcon, StopIcon } from 'components/icon/icon';
import { Process, isProcessRunnable, isProcessResumable, isProcessCancelable } from 'store/processes/process';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { ProcessDetailsAttributes } from './process-details-attributes';
import { ProcessStatus } from 'views-components/data-explorer/renderers';
import classNames from 'classnames';

type CssRules = 'card' | 'content' | 'title' | 'header' | 'cancelButton' | 'avatar' | 'iconHeader' | 'actionButton';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing.unit * 0.5
    },
    content: {
        padding: theme.spacing.unit * 1.0,
        paddingTop: theme.spacing.unit * 0.5,
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 1,
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5,
        color: theme.customs.colors.green700,
    },
    actionButton: {
        padding: "0px 5px 0 0",
        marginRight: "5px",
        fontSize: '0.78rem',
    },
    cancelButton: {
        color: theme.palette.common.white,
        backgroundColor: theme.customs.colors.red900,
        '&:hover': {
            backgroundColor: theme.customs.colors.red900,
        },
        '& svg': {
            fontSize: '22px',
        },
    },
});

export interface ProcessDetailsCardDataProps {
    process: Process;
    cancelProcess: (uuid: string) => void;
    startProcess: (uuid: string) => void;
    resumeOnHoldWorkflow: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

type ProcessDetailsCardProps = ProcessDetailsCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessDetailsCard = withStyles(styles)(
    ({ cancelProcess, startProcess, resumeOnHoldWorkflow, onContextMenu, classes, process, doHidePanel, panelName }: ProcessDetailsCardProps) => {
        let runAction: ((uuid: string) => void) | undefined = undefined;
        if (isProcessRunnable(process)) {
            runAction = startProcess;
        } else if (isProcessResumable(process)) {
            runAction = resumeOnHoldWorkflow;
        }

        return <Card className={classes.card}>
            <CardHeader
                className={classes.header}
                classes={{
                    content: classes.title,
                    avatar: classes.avatar,
                }}
                avatar={<ProcessIcon className={classes.iconHeader} />}
                title={
                    <Tooltip title={process.containerRequest.name} placement="bottom-start">
                        <Typography noWrap variant='h6'>
                            {process.containerRequest.name}
                        </Typography>
                    </Tooltip>
                }
                subheader={
                    <Tooltip title={getDescription(process)} placement="bottom-start">
                        <Typography noWrap variant='body1' color='inherit'>
                            {getDescription(process)}
                        </Typography>
                    </Tooltip>}
                action={
                    <div>
                        {runAction !== undefined &&
                            <Button
                                data-cy="process-run-button"
                                variant="contained"
                                size="small"
                                color="primary"
                                className={classes.actionButton}
                                onClick={() => runAction && runAction(process.containerRequest.uuid)}>
                                <StartIcon />
                                Run
                            </Button>}
                        {isProcessCancelable(process) &&
                            <Button
                                data-cy="process-cancel-button"
                                variant="contained"
                                size="small"
                                color="primary"
                                className={classNames(classes.actionButton, classes.cancelButton)}
                                onClick={() => cancelProcess(process.containerRequest.uuid)}>
                                <StopIcon />
                                Cancel
                            </Button>}
                        <ProcessStatus uuid={process.containerRequest.uuid} />
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton
                                aria-label="More options"
                                onClick={event => onContextMenu(event)}>
                                <MoreVerticalIcon />
                            </IconButton>
                        </Tooltip>
                        {doHidePanel &&
                            <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                                <IconButton onClick={doHidePanel}><CloseIcon /></IconButton>
                            </Tooltip>}
                    </div>
                } />
            <CardContent className={classes.content}>
                <ProcessDetailsAttributes request={process.containerRequest} twoCol hideProcessPanelRedundantFields />
            </CardContent>
        </Card>;
    }
);

const getDescription = (process: Process) =>
    process.containerRequest.description || '(no-description)';
