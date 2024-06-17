// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Tooltip } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CProgressStacked, CProgress } from '@coreui/react';
import { useAsyncInterval } from "common/use-async-interval";
import { Process, isProcessRunning } from "store/processes/process";
import { connect } from "react-redux";
import { Dispatch } from "redux";
import { fetchSubprocessProgress } from "store/subprocess-panel/subprocess-panel-actions";
import { ProcessStatusFilter } from "store/resource-type-filters/resource-type-filters";

type CssRules = 'progressWrapper' | 'progressStacked';

const styles: CustomStyleRulesCallback<CssRules> = (theme) => ({
    progressWrapper: {
        margin: "28px 0 0",
        flexGrow: 1,
        flexBasis: "100px",
    },
    progressStacked: {
        border: "1px solid gray",
        height: "10px",
        // Override stripe color to be close to white
        "& .progress-bar-striped": {
            backgroundImage:
                "linear-gradient(45deg,rgba(255,255,255,.80) 25%,transparent 25%,transparent 50%,rgba(255,255,255,.80) 50%,rgba(255,255,255,.80) 75%,transparent 75%,transparent)",
        },
    },
});

export interface ProgressBarDataProps {
    process: Process;
}

export interface ProgressBarActionProps {
    fetchSubprocessProgress: (requestingContainerUuid: string) => Promise<ProgressBarData | undefined>;
}

type ProgressBarProps = ProgressBarDataProps & ProgressBarActionProps & WithStyles<CssRules>;

export type ProgressBarData = {
    [ProcessStatusFilter.COMPLETED]: number;
    [ProcessStatusFilter.RUNNING]: number;
    [ProcessStatusFilter.FAILED]: number;
    [ProcessStatusFilter.QUEUED]: number;
};

const mapDispatchToProps = (dispatch: Dispatch): ProgressBarActionProps => ({
    fetchSubprocessProgress: (requestingContainerUuid: string) => {
        return dispatch<any>(fetchSubprocessProgress(requestingContainerUuid));
    },
});

export const SubprocessProgressBar = connect(null, mapDispatchToProps)(withStyles(styles)(
    ({ process, classes, fetchSubprocessProgress }: ProgressBarProps) => {

        const [progressData, setProgressData] = useState<ProgressBarData | undefined>(undefined);
        const requestingContainerUuid = process.containerRequest.containerUuid;
        const isRunning = isProcessRunning(process);

        useAsyncInterval(async () => (
            requestingContainerUuid && setProgressData(await fetchSubprocessProgress(requestingContainerUuid))
        ), isRunning ? 5000 : null);

        useEffect(() => {
            if (!isRunning && requestingContainerUuid) {
                fetchSubprocessProgress(requestingContainerUuid)
                    .then(result => setProgressData(result));
            }
        }, [fetchSubprocessProgress, isRunning, requestingContainerUuid]);

        let tooltip = "";
        if (progressData) {
            let total = 0;
            [ProcessStatusFilter.COMPLETED,
            ProcessStatusFilter.RUNNING,
            ProcessStatusFilter.FAILED,
            ProcessStatusFilter.QUEUED].forEach(psf => {
                if (progressData[psf] > 0) {
                    if (tooltip.length > 0) { tooltip += ", "; }
                    tooltip += `${progressData[psf]} ${psf}`;
                    total += progressData[psf];
                }
            });
            if (total > 0) {
                if (tooltip.length > 0) { tooltip += ", "; }
                tooltip += `${total} Total`;
            }
        }

        return progressData !== undefined && getStatusTotal(progressData) > 0 ? <div className={classes.progressWrapper}>
            <Tooltip title={tooltip}>
                <CProgressStacked className={classes.progressStacked}>
                    <CProgress height={10} color="success"
                        value={getStatusPercent(progressData, ProcessStatusFilter.COMPLETED)} />
                    <CProgress height={10} color="success" variant="striped"
                        value={getStatusPercent(progressData, ProcessStatusFilter.RUNNING)} />
                    <CProgress height={10} color="danger"
                        value={getStatusPercent(progressData, ProcessStatusFilter.FAILED)} />
                    <CProgress height={10} color="secondary" variant="striped"
                        value={getStatusPercent(progressData, ProcessStatusFilter.QUEUED)} />
                </CProgressStacked>
            </Tooltip>
        </div> : <></>;
    }
));

const getStatusTotal = (progressData: ProgressBarData) =>
    (Object.keys(progressData).reduce((accumulator, key) => (accumulator += progressData[key]), 0));

/**
 * Gets the integer percent value for process status
 */
const getStatusPercent = (progressData: ProgressBarData, status: keyof ProgressBarData) =>
    (progressData[status] / getStatusTotal(progressData) * 100);
