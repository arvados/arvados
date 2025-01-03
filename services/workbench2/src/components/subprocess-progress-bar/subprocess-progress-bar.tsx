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
import { fetchProcessProgressBarStatus, isProcess } from "store/subprocess-panel/subprocess-panel-actions";
import { ProcessStatusFilter, serializeOnlyProcessTypeFilters } from "store/resource-type-filters/resource-type-filters";
import { ProjectResource } from "models/project";
import { getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { RootState } from "store/store";
import { ProcessResource } from "models/process";
import { getDataExplorerColumnFilters } from "store/data-explorer/data-explorer-middleware-service";
import { ProjectPanelRunColumnNames } from "views/project-panel/project-panel-run";
import { DataColumns } from "components/data-table/data-column";

type CssRules = 'progressStacked';

const styles: CustomStyleRulesCallback<CssRules> = (theme) => ({
    progressWrapper: {
        margin: "28px 0 0",
        flexGrow: 1,
        flexBasis: "100px",
    },
    progressStacked: {
        border: "1px solid gray",
        height: "10px",
        marginTop: "-5px",
        // Override stripe color to be close to white
        "& .progress-bar-striped": {
            backgroundImage:
                "linear-gradient(45deg,rgba(255,255,255,.80) 25%,transparent 25%,transparent 50%,rgba(255,255,255,.80) 50%,rgba(255,255,255,.80) 75%,transparent 75%,transparent)",
        },
    },
});

export interface ProgressBarDataProps {
    parentResource: Process | ProjectResource | undefined;
    dataExplorerId?: string;
    typeFilter?: string;
}

export interface ProgressBarActionProps {
    fetchProcessProgressBarStatus: (parentResourceUuid: string, typeFilter?: string) => Promise<ProgressBarStatus | undefined>;
}

type ProgressBarProps = ProgressBarDataProps & ProgressBarActionProps & WithStyles<CssRules>;

export type ProgressBarCounts = {
    [ProcessStatusFilter.COMPLETED]: number;
    [ProcessStatusFilter.RUNNING]: number;
    [ProcessStatusFilter.FAILED]: number;
    [ProcessStatusFilter.QUEUED]: number;
};

export type ProgressBarStatus = {
    counts: ProgressBarCounts;
    shouldPollProject: boolean;
};

const mapStateToProps = (state: RootState, props: ProgressBarDataProps) => {
    let typeFilter: string | undefined = undefined;

    if (props.dataExplorerId) {
        const dataExplorerState = getDataExplorer(state.dataExplorer, props.dataExplorerId);
        const columns = dataExplorerState.columns as DataColumns<ProcessResource>;
        typeFilter = serializeOnlyProcessTypeFilters(false)(getDataExplorerColumnFilters(columns, ProjectPanelRunColumnNames.TYPE));
    }

    return { typeFilter };
};

const mapDispatchToProps = (dispatch: Dispatch): ProgressBarActionProps => ({
    fetchProcessProgressBarStatus: (parentResourceUuid: string, typeFilter?: string) => {
        return dispatch<any>(fetchProcessProgressBarStatus(parentResourceUuid, typeFilter));
    },
});

export const SubprocessProgressBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    ({ parentResource, typeFilter, classes, fetchProcessProgressBarStatus }: ProgressBarProps) => {

        const [progressCounts, setProgressData] = useState<ProgressBarCounts | undefined>(undefined);
        const [shouldPollProject, setShouldPollProject] = useState<boolean>(false);
        const shouldPollProcess = isProcess(parentResource) ? isProcessRunning(parentResource) : false;

        // Should polling be active based on container status
        // or result of aggregated project process contents
        const shouldPoll = shouldPollProject || shouldPollProcess;

        const parentUuid = parentResource
            ? isProcess(parentResource)
                ? parentResource.containerRequest.uuid
                : parentResource.uuid
            : "";

        // Runs periodically whenever polling should be happeing
        // Either when the workflow is running (shouldPollProcess) or when the
        //   project contains steps in an active state (shouldPollProject)
        useAsyncInterval(async () => {
            if (parentUuid) {
                fetchProcessProgressBarStatus(parentUuid, typeFilter)
                    .then(result => {
                        if (result) {
                            setProgressData(result.counts);
                            setShouldPollProject(result.shouldPollProject);
                        }
                    });
            }
        }, shouldPoll ? 5000 : null);

        // Runs fetch on first load for processes and projects, except when
        //   process is running since polling will be enabled by shouldPoll.
        // Project polling starts false so this is still needed for project
        //   initial load to set shouldPollProject and kick off shouldPoll
        // Watches shouldPollProcess but not shouldPollProject
        //   * This runs a final fetch when process ends & is updated through
        //     websocket / store
        //   * We ignore shouldPollProject entirely since it changes to false
        //     as a result of a fetch so the data is already up to date
        useEffect(() => {
            if (!shouldPollProcess && parentUuid) {
                fetchProcessProgressBarStatus(parentUuid, typeFilter)
                    .then(result => {
                        if (result) {
                            setProgressData(result.counts);
                            setShouldPollProject(result.shouldPollProject);
                        }
                    });
            }
        }, [fetchProcessProgressBarStatus, shouldPollProcess, parentUuid, typeFilter]);

        let tooltip = "";
        if (progressCounts) {
            let total = 0;
            [ProcessStatusFilter.COMPLETED,
            ProcessStatusFilter.RUNNING,
            ProcessStatusFilter.FAILED,
            ProcessStatusFilter.QUEUED].forEach(psf => {
                if (progressCounts[psf] > 0) {
                    if (tooltip.length > 0) { tooltip += ", "; }
                    tooltip += `${progressCounts[psf]} ${psf}`;
                    total += progressCounts[psf];
                }
            });
            if (total > 0) {
                if (tooltip.length > 0) { tooltip += ", "; }
                tooltip += `${total} Total`;
            }
        }

        return progressCounts !== undefined && getStatusTotal(progressCounts) > 0 ? <Tooltip title={tooltip}>
            <CProgressStacked className={classes.progressStacked}>
                <CProgress height={10} color="success"
                    value={getStatusPercent(progressCounts, ProcessStatusFilter.COMPLETED)} />
                <CProgress height={10} color="success" variant="striped"
                    value={getStatusPercent(progressCounts, ProcessStatusFilter.RUNNING)} />
                <CProgress height={10} color="danger"
                    value={getStatusPercent(progressCounts, ProcessStatusFilter.FAILED)} />
                <CProgress height={10} color="secondary" variant="striped"
                    value={getStatusPercent(progressCounts, ProcessStatusFilter.QUEUED)} />
            </CProgressStacked>
        </Tooltip> : <></>;
    }
));

const getStatusTotal = (progressCounts: ProgressBarCounts) =>
    (Object.keys(progressCounts).reduce((accumulator, key) => (accumulator += progressCounts[key]), 0));

/**
 * Gets the integer percent value for process status
 */
const getStatusPercent = (progressCounts: ProgressBarCounts, status: keyof ProgressBarCounts) =>
    (progressCounts[status] / getStatusTotal(progressCounts) * 100);
