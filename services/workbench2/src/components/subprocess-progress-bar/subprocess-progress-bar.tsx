// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from "react";
import { StyleRulesCallback, Tooltip, WithStyles, withStyles } from "@material-ui/core";
import { CProgressStacked, CProgress } from '@coreui/react';
import { useAsyncInterval } from "common/use-async-interval";
import { Process } from "store/processes/process";
import { connect } from "react-redux";
import { Dispatch } from "redux";
import { fetchProcessProgressBarStatus } from "store/subprocess-panel/subprocess-panel-actions";
import { ProcessStatusFilter, serializeOnlyProcessTypeFilters } from "store/resource-type-filters/resource-type-filters";
import { ProjectResource } from "models/project";
import { getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { RootState } from "store/store";
import { ProcessResource } from "models/process";
import { getDataExplorerColumnFilters } from "store/data-explorer/data-explorer-middleware-service";
import { ProjectPanelRunColumnNames } from "views/project-panel/project-panel-run";
import { DataColumns } from "components/data-table/data-table";

type CssRules = 'progressWrapper' | 'progressStacked';

const styles: StyleRulesCallback<CssRules> = (theme) => ({
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
    parentResource: Process | ProjectResource | undefined;
    dataExplorerId?: string;
    typeFilter?: string;
}

export interface ProgressBarActionProps {
    fetchProcessProgressBarStatus: (parentResource: Process | ProjectResource, typeFilter?: string) => Promise<ProgressBarStatus | undefined>;
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
    isRunning: boolean;
};

const mapStateToProps = (state: RootState, props: ProgressBarDataProps) => {
    let typeFilter: string | undefined = undefined;

    if (props.dataExplorerId) {
        const dataExplorerState = getDataExplorer(state.dataExplorer, props.dataExplorerId);
        const columns = dataExplorerState.columns as DataColumns<string, ProcessResource>;
        typeFilter = serializeOnlyProcessTypeFilters(getDataExplorerColumnFilters(columns, ProjectPanelRunColumnNames.TYPE));
    }

    return { typeFilter };
};

const mapDispatchToProps = (dispatch: Dispatch): ProgressBarActionProps => ({
    fetchProcessProgressBarStatus: (parentResource: Process | ProjectResource, typeFilter?: string) => {
        return dispatch<any>(fetchProcessProgressBarStatus(parentResource, typeFilter));
    },
});

export const SubprocessProgressBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    ({ parentResource, typeFilter, classes, fetchProcessProgressBarStatus }: ProgressBarProps) => {

        const [progressCounts, setProgressData] = useState<ProgressBarCounts | undefined>(undefined);
        const [isRunning, setIsRunning] = useState<boolean>(false);

        useAsyncInterval(async () => {
            if (parentResource) {
                fetchProcessProgressBarStatus(parentResource, typeFilter)
                    .then(result => {
                        if (result) {
                            setProgressData(result.counts);
                            setIsRunning(result.isRunning);
                        }
                    });
            }
        }, isRunning ? 5000 : null);

        useEffect(() => {
            if (!isRunning && parentResource) {
                fetchProcessProgressBarStatus(parentResource, typeFilter)
                    .then(result => {
                        if (result) {
                            setProgressData(result.counts);
                            setIsRunning(result.isRunning);
                        }
                    });
            }
        }, [fetchProcessProgressBarStatus, isRunning, parentResource, typeFilter]);

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

        return progressCounts !== undefined && getStatusTotal(progressCounts) > 0 ? <div className={classes.progressWrapper}>
            <Tooltip title={tooltip}>
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
            </Tooltip>
        </div> : <></>;
    }
));

const getStatusTotal = (progressCounts: ProgressBarCounts) =>
    (Object.keys(progressCounts).reduce((accumulator, key) => (accumulator += progressCounts[key]), 0));

/**
 * Gets the integer percent value for process status
 */
const getStatusPercent = (progressCounts: ProgressBarCounts, status: keyof ProgressBarCounts) =>
    (progressCounts[status] / getStatusTotal(progressCounts) * 100);
