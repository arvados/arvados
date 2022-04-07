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

type CssRules = 'heading' | 'summary' | 'details' | 'error' | 'errorColor' | 'warning' | 'warningColor';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    heading: {
        fontSize: '1rem',
    },
    summary: {
        paddingLeft: theme.spacing.unit * 1,
        paddingRight: theme.spacing.unit * 1,
    },
    details: {
        paddingLeft: theme.spacing.unit * 1,
        paddingRight: theme.spacing.unit * 1,
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
});
export interface ProcessRuntimeStatusDataProps {
    runtimeStatus: RuntimeStatus | undefined;
}

type ProcessRuntimeStatusProps = ProcessRuntimeStatusDataProps & WithStyles<CssRules>;

export const ProcessRuntimeStatus = withStyles(styles)(
    ({ runtimeStatus, classes }: ProcessRuntimeStatusProps) => {
    return <>
        { runtimeStatus?.error &&
        <ExpansionPanel className={classes.error} elevation={0}>
            <ExpansionPanelSummary className={classes.summary} expandIcon={<ExpandMoreIcon />}>
                <Typography className={classNames(classes.heading, classes.errorColor)}>
                    {`Error: ${runtimeStatus.error }`}
                </Typography>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.details}>
                <Typography className={classes.errorColor}>
                    {runtimeStatus?.errorDetail || 'No additional error details available'}
                </Typography>
            </ExpansionPanelDetails>
        </ExpansionPanel>
        }
        { runtimeStatus?.warning &&
        <ExpansionPanel className={classes.warning} elevation={0}>
            <ExpansionPanelSummary className={classes.summary} expandIcon={<ExpandMoreIcon />}>
                <Typography className={classNames(classes.heading, classes.warningColor)}>
                    {`Warning: ${runtimeStatus.warning }`}
                </Typography>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.details}>
                <Typography className={classes.warningColor}>
                    {runtimeStatus?.warningDetail || 'No additional warning details available'}
                </Typography>
            </ExpansionPanelDetails>
        </ExpansionPanel>
        }
    </>
});