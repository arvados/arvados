// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    ExpansionPanel,
    ExpansionPanelDetails,
    ExpansionPanelSummary,
    StyleRulesCallback,
    Typography,
    withStyles,
    WithStyles
} from "@material-ui/core";
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import { RuntimeStatus } from "models/runtime-status";
import { ArvadosTheme } from 'common/custom-theme';
import classNames from 'classnames';

type CssRules = 'root'
    | 'heading'
    | 'summary'
    | 'summaryText'
    | 'details'
    | 'detailsText'
    | 'error'
    | 'errorColor'
    | 'warning'
    | 'warningColor'
    | 'disabledPanelSummary';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        marginBottom: theme.spacing.unit * 1,
    },
    heading: {
        fontSize: '1rem',
    },
    summary: {
        paddingLeft: theme.spacing.unit * 1,
        paddingRight: theme.spacing.unit * 1,
    },
    summaryText: {
        whiteSpace: 'pre-line',
    },
    details: {
        paddingLeft: theme.spacing.unit * 1,
        paddingRight: theme.spacing.unit * 1,
    },
    detailsText: {
        fontSize: '0.8rem',
        marginTop: '0px',
        marginBottom: '0px',
        whiteSpace: 'pre-line',
    },
    errorColor: {
        color: theme.customs.colors.red900,
    },
    error: {
        backgroundColor: theme.customs.colors.red100,

    },
    warning: {
        backgroundColor: theme.customs.colors.yellow100,
    },
    warningColor: {
        color: theme.customs.colors.yellow900,
    },
    disabledPanelSummary: {
        cursor: 'default',
        pointerEvents: 'none',
    }
});
export interface ProcessRuntimeStatusDataProps {
    runtimeStatus: RuntimeStatus | undefined;
    containerCount: number;
}

type ProcessRuntimeStatusProps = ProcessRuntimeStatusDataProps & WithStyles<CssRules>;

export const ProcessRuntimeStatus = withStyles(styles)(
    ({ runtimeStatus, containerCount, classes }: ProcessRuntimeStatusProps) => {
    return <div className={classes.root}>
        { runtimeStatus?.error &&
        <div data-cy='process-runtime-status-error'><ExpansionPanel className={classes.error} elevation={0}>
            <ExpansionPanelSummary className={classNames(classes.summary, classes.detailsText)} expandIcon={<ExpandMoreIcon />}>
                <Typography className={classNames(classes.heading, classes.errorColor)}>
                    {`Error: ${runtimeStatus.error }`}
                </Typography>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.details}>
                <Typography className={classNames(classes.errorColor, classes.detailsText)}>
                    {runtimeStatus?.errorDetail || 'No additional error details available'}
                </Typography>
            </ExpansionPanelDetails>
        </ExpansionPanel></div>
        }
        { runtimeStatus?.warning &&
        <div data-cy='process-runtime-status-warning' ><ExpansionPanel className={classes.warning} elevation={0}>
            <ExpansionPanelSummary className={classNames(classes.summary, classes.detailsText)} expandIcon={<ExpandMoreIcon />}>
                <Typography className={classNames(classes.heading, classes.warningColor)}>
                    {`Warning: ${runtimeStatus.warning }`}
                </Typography>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.details}>
                <Typography className={classNames(classes.warningColor, classes.detailsText)}>
                    {runtimeStatus?.warningDetail || 'No additional warning details available'}
                </Typography>
            </ExpansionPanelDetails>
        </ExpansionPanel></div>
        }
        { containerCount > 1 &&
        <div data-cy='process-runtime-status-warning' ><ExpansionPanel className={classes.warning} elevation={0} expanded={false}>
            <ExpansionPanelSummary className={classNames(classes.summary, classes.detailsText, classes.disabledPanelSummary)}>
                <Typography className={classNames(classes.heading, classes.warningColor)}>
                    {`Warning: Process retried ${containerCount - 1} time${containerCount > 2 ? 's' : ''} due to failure.`}
                </Typography>
            </ExpansionPanelSummary>
        </ExpansionPanel></div>
        }
    </div>
});
