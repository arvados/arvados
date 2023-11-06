// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from "react";
import { StyleRulesCallback, Typography, WithStyles, withStyles } from "@material-ui/core";
import { CProgressStacked, CProgress } from '@coreui/react';
import { useAsyncInterval } from "common/use-async-interval";
import { Process, isProcessRunning } from "store/processes/process";
import { connect } from "react-redux";
import { Dispatch } from "redux";
import { fetchSubprocessProgress } from "store/subprocess-panel/subprocess-panel-actions";
import { ProcessStatusFilter } from "store/resource-type-filters/resource-type-filters";

type CssRules = 'progressWrapper' | 'progressStacked' ;

const styles: StyleRulesCallback<CssRules> = (theme) => ({
    progressWrapper: {
        margin: "0 20px",
    },
    progressStacked: {
        border: "1px solid gray",
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
    ({process, classes, fetchSubprocessProgress}: ProgressBarProps) => {

        const [progressData, setProgressData] = useState<ProgressBarData|undefined>(undefined);
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

        return progressData !== undefined && getStatusTotal(progressData) > 0 ? <div className={classes.progressWrapper}>
            <CProgressStacked className={classes.progressStacked}>
                <CProgress height={20} color="success" title="Completed"
                    value={getStatusPercent(progressData, ProcessStatusFilter.COMPLETED)} />
                <CProgress height={20} color="success" title="Running" variant="striped" animated
                    value={getStatusPercent(progressData, ProcessStatusFilter.RUNNING)} />
                <CProgress height={20} color="danger" title="Failed"
                    value={getStatusPercent(progressData, ProcessStatusFilter.FAILED)} />
                <CProgress height={20} color="secondary" title="Queued" variant="striped" animated
                    value={getStatusPercent(progressData, ProcessStatusFilter.QUEUED)} />
            </CProgressStacked>
            <Typography variant="body2">
                {progressData[ProcessStatusFilter.COMPLETED]} Completed, {progressData[ProcessStatusFilter.RUNNING]} Running, {progressData[ProcessStatusFilter.FAILED]} Failed, {progressData[ProcessStatusFilter.QUEUED]} Queued
            </Typography>
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
