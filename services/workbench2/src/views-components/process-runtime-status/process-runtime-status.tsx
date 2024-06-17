// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Accordion, AccordionDetails, AccordionSummary, Paper, Typography } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
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
    | 'paperRoot';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        marginBottom: theme.spacing(1),
    },
    heading: {
        fontSize: '1rem',
    },
    summary: {
        paddingLeft: theme.spacing(1),
        paddingRight: theme.spacing(1),
    },
    summaryText: {
        whiteSpace: 'pre-line',
    },
    details: {
        paddingLeft: theme.spacing(1),
        paddingRight: theme.spacing(1),
    },
    detailsText: {
        fontSize: '0.8rem',
        marginTop: '0px',
        marginBottom: '0px',
        whiteSpace: 'pre-line',
    },
    errorColor: {
        color: theme.customs.colors.grey700,
    },
    error: {
        backgroundColor: theme.customs.colors.red100,

    },
    warning: {
        backgroundColor: theme.customs.colors.yellow100,
    },
    warningColor: {
        color: theme.customs.colors.grey700,
    },
    paperRoot: {
        minHeight: theme.spacing(6),
        display: 'flex',
        alignItems: 'center',
    },
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
        <div data-cy='process-runtime-status-error'><Accordion className={classes.error} elevation={0}>
            <AccordionSummary className={classNames(classes.summary, classes.detailsText)} expandIcon={<ExpandMoreIcon />}>
                <Typography className={classNames(classes.heading, classes.errorColor)}>
                    {`Error: ${runtimeStatus.error }`}
                </Typography>
            </AccordionSummary>
            <AccordionDetails className={classes.details}>
                <Typography className={classNames(classes.errorColor, classes.detailsText)}>
                    {runtimeStatus?.errorDetail || 'No additional error details available'}
                </Typography>
            </AccordionDetails>
        </Accordion></div>
        }
        { runtimeStatus?.warning &&
        <div data-cy='process-runtime-status-warning' ><Accordion className={classes.warning} elevation={0}>
            <AccordionSummary className={classNames(classes.summary, classes.detailsText)} expandIcon={<ExpandMoreIcon />}>
                <Typography className={classNames(classes.heading, classes.warningColor)}>
                    {`Warning: ${runtimeStatus.warning }`}
                </Typography>
            </AccordionSummary>
            <AccordionDetails className={classes.details}>
                <Typography className={classNames(classes.warningColor, classes.detailsText)}>
                    {runtimeStatus?.warningDetail || 'No additional warning details available'}
                </Typography>
            </AccordionDetails>
        </Accordion></div>
        }
        { containerCount > 1 &&
        <div data-cy='process-runtime-status-retry-warning' >
            <Paper className={classNames(classes.warning, classes.paperRoot)} elevation={0}>
                <Typography className={classNames(classes.heading, classes.summary, classes.warningColor)}>
                    {`Warning: Process retried ${containerCount - 1} time${containerCount > 2 ? 's' : ''} due to failure.`}
                </Typography>
            </Paper>
        </div>
        }
    </div>
});
