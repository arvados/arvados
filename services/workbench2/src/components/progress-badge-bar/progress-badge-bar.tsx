// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, {useEffect, useState, useRef} from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ProcessStatusSortButton } from "components/process-status-badge/process-status-badge";
import { CustomStyleRulesCallback } from "common/custom-theme";
import { ArvadosTheme } from "common/custom-theme";
import { Process, isProcessRunning } from "store/processes/process";
import { ProjectResource } from "models/project";
import { DataExplorerState, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { RootState } from "store/store";
import { DataTableFilters } from 'components/data-table-filters/data-table-filters';
import { dataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { ProcessStatusFilter, serializeOnlyProcessTypeFilters } from "store/resource-type-filters/resource-type-filters";
import { ProjectPanelRunColumnNames } from "views/project-panel/project-panel-run";
import { DataColumns } from "components/data-table/data-column";
import { fetchProcessProgressBarStatus, isProcess, ProgressBadgeCounts } from "store/subprocess-panel/subprocess-panel-actions";
import { ProcessResource } from "models/process";
import { getDataExplorerColumnFilters } from "store/data-explorer/data-explorer-middleware-service";
import { useAsyncInterval } from "common/use-async-interval";

type CssRules = 'root' | 'button';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
        justifyContent: 'flex-start',
        marginLeft: '-8px',
    },
    button: {
        marginRight: theme.spacing(1),
    },
});

type ProgressBarStatus = {
    counts: ProgressBadgeCounts;
    shouldPollProject: boolean;
};

type ProgressBadgeBarDataProps = {
    parentResource?: Process | ProjectResource;
    dataExplorer: DataExplorerState;
    dataExplorerId?: string;
    pathName?: string;
};

type ProgressBadgeBarActionProps = {
    onFiltersChange: (filters: DataTableFilters, columnName: string, id: string, status: string) => void;
    fetchProcessProgressBarStatus: (parentResourceUuid: string, typeFilter?: string) => Promise<ProgressBarStatus | undefined>;
};

type ProgressBadgeBarProps = ProgressBadgeBarDataProps & ProgressBadgeBarActionProps & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): Pick<ProgressBadgeBarDataProps, 'dataExplorer' | 'pathName'> => {
    return {
        dataExplorer: state.dataExplorer,
        pathName: state.router.location?.pathname,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProgressBadgeBarActionProps => ({
    onFiltersChange: (filters: DataTableFilters, columnName: string, id: string, status: string) => {
        const selectedStatusFilters = selectStatus(status, filters);
        dispatch(dataExplorerActions.SET_FILTERS({ id, columnName, filters: selectedStatusFilters }));
    },
    fetchProcessProgressBarStatus: (parentResourceUuid: string, typeFilter?: string) => {
        return dispatch<any>(fetchProcessProgressBarStatus(parentResourceUuid, typeFilter));
    },
});


export const ProgressBadgeBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    ({ parentResource, classes, dataExplorer, dataExplorerId, pathName, onFiltersChange, fetchProcessProgressBarStatus }: ProgressBadgeBarProps ) => {

        const [progressCounts, setProgressData] = useState<ProgressBadgeCounts | undefined>(undefined);
        const [shouldPollProject, setShouldPollProject] = useState<boolean>(false);
        const shouldPollProcess = isProcess(parentResource) ? isProcessRunning(parentResource) : false;
        const statusColumn = getDataExplorer(dataExplorer, dataExplorerId || '').columns.find(column => column.name === 'Status');
        const filterLabels: string[] = statusColumn ? Object.keys(statusColumn.filters) : [];

        let typeFilter = useRef('');

        useEffect(() => {
            if (dataExplorerId) {
                const dataExplorerState = getDataExplorer(dataExplorer, dataExplorerId);
                const columns = dataExplorerState.columns as DataColumns<string, ProcessResource>;
                typeFilter.current = serializeOnlyProcessTypeFilters(false)(getDataExplorerColumnFilters(columns, ProjectPanelRunColumnNames.TYPE));
            }
        }, [dataExplorer, dataExplorerId]);

        //reset filters when path changes
        useEffect(() => {
            if (statusColumn && dataExplorerId) {
                onFiltersChange(statusColumn?.filters, statusColumn?.name, dataExplorerId, ProcessStatusFilter.ALL);
            }
            // eslint-disable-next-line react-hooks/exhaustive-deps
        }, [pathName]);

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
            if (parentUuid && typeFilter.current) {
                fetchProcessProgressBarStatus(parentUuid, typeFilter.current)
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
                fetchProcessProgressBarStatus(parentUuid, typeFilter.current)
                    .then(result => {
                        if (result) {
                            setProgressData(result.counts);
                            setShouldPollProject(result.shouldPollProject);
                        }
                    });
            }
        }, [fetchProcessProgressBarStatus, shouldPollProcess, parentUuid, typeFilter, dataExplorer]);

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

    return statusColumn && dataExplorerId ? (
    <div className={classes.root}>
        {filterLabels.map(status =>
            <div key={status} className={classes.button}>
            <ProcessStatusSortButton
                status={status}
                onFiltersChange={onFiltersChange}
                filters={statusColumn.filters}
                columnName={statusColumn.name}
                dataExplorerId={dataExplorerId}
                numProcesses={getStatusTotal(progressCounts, status)}
                />
                </div>
        )}
    </div>
    ) : (
        <div>-</div>
    )}
));

function selectStatus(status: string, filters: DataTableFilters) {
    const filterCopy = { ...filters };
    for (const key in filterCopy) {
        if (filterCopy[key].selected === true) {
            filterCopy[key].selected = false;
        }
        if (filterCopy[key].id === status) {
            filterCopy[key].selected = true;
        }
    }
    return filterCopy;
}

const getStatusTotal = (progressCounts: ProgressBadgeCounts | undefined, status: string) => {
    return progressCounts?.[status] || 0;
}