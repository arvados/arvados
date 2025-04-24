// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ProcessStatusSortButton } from "components/process-status-badge/process-status-badge";
import { CustomStyleRulesCallback } from "common/custom-theme";
import { ArvadosTheme } from "common/custom-theme";
import { Process } from "store/processes/process";
import { ProjectResource } from "models/project";
import { DataExplorerState } from "store/data-explorer/data-explorer-reducer";
import { RootState } from "store/store";
import { DataTableFilters } from 'components/data-table-filters/data-table-filters';
import { dataExplorerActions } from 'store/data-explorer/data-explorer-action';

type CssRules = 'root';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
        justifyContent: 'space-between',
    },
});

type ProgressBadgeBarDataProps = {
    parentResource: Process | ProjectResource | undefined;
        dataExplorer: DataExplorerState;
        dataExplorerId?: string;
};

type ProgressBadgeBarActionProps = {
    onFiltersChange: (filters: DataTableFilters, columnName: string, id: string) => void;
};

type ProgressBadgeBarProps = ProgressBadgeBarDataProps & ProgressBadgeBarActionProps & WithStyles<CssRules>;

const mapStateToProps = (state: RootState) => {
    return { dataExplorer: state.dataExplorer };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onFiltersChange: (filters: DataTableFilters, columnName: string, id: string, status: string) => {
        const selectedStatusFilters = selectStatus(status, filters);
        dispatch(dataExplorerActions.SET_FILTERS({ id, columnName, filters: selectedStatusFilters }));
    },
});


export const ProgressBadgeBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    ({ parentResource, classes, dataExplorer, dataExplorerId, onFiltersChange }: ProgressBadgeBarProps ) => {

    const statusColumn = dataExplorer[dataExplorerId || ''].columns.find(column => column.name === 'Status');
    const filterLabels: string[] = statusColumn ? Object.keys(statusColumn.filters) : [];

    return statusColumn && dataExplorerId ? (
    <div className={classes.root}>
        {filterLabels.map(status =>
            <ProcessStatusSortButton
                status={status}
                onFiltersChange={onFiltersChange}
                filters={statusColumn.filters}
                columnName={statusColumn.name}
                dataExplorerId={dataExplorerId}
                />
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
