// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "store/store";
import { DataExplorer as DataExplorerComponent } from "components/data-explorer/data-explorer";
import { getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { Dispatch } from "redux";
import { dataExplorerActions } from "store/data-explorer/data-explorer-action";
import { DataColumn } from "components/data-table/data-column";
import { DataColumns } from "components/data-table/data-table";
import { DataTableFilters } from 'components/data-table-filters/data-table-filters-tree';
import { LAST_REFRESH_TIMESTAMP } from "components/refresh-button/refresh-button";

interface Props {
    id: string;
    onRowClick: (item: any) => void;
    onContextMenu?: (event: React.MouseEvent<HTMLElement>, item: any, isAdmin?: boolean) => void;
    onRowDoubleClick: (item: any) => void;
    extractKey?: (item: any) => React.Key;
}

const mapStateToProps = (state: RootState, { id }: Props) => {
    const progress = state.progressIndicator.find(p => p.id === id);
    const dataExplorerState = getDataExplorer(state.dataExplorer, id);
    const currentRoute = state.router.location ? state.router.location.pathname : '';
    const currentRefresh = localStorage.getItem(LAST_REFRESH_TIMESTAMP) || '';
    const currentItemUuid = currentRoute === '/workflows' ? state.properties.workflowPanelDetailsUuid : state.detailsPanel.resourceUuid;
    return {
        ...dataExplorerState,
        working: !!progress?.working,
        currentRefresh: currentRefresh,
        currentRoute: currentRoute,
        paperKey: currentRoute,
        currentItemUuid,
    };
};

const mapDispatchToProps = () => {
    return (dispatch: Dispatch, { id, onRowClick, onRowDoubleClick, onContextMenu }: Props) => ({
        onSetColumns: (columns: DataColumns<any>) => {
            dispatch(dataExplorerActions.SET_COLUMNS({ id, columns }));
        },

        onSearch: (searchValue: string) => {
            dispatch(dataExplorerActions.SET_EXPLORER_SEARCH_VALUE({ id, searchValue }));
        },

        onColumnToggle: (column: DataColumn<any>) => {
            dispatch(dataExplorerActions.TOGGLE_COLUMN({ id, columnName: column.name }));
        },

        onSortToggle: (column: DataColumn<any>) => {
            dispatch(dataExplorerActions.TOGGLE_SORT({ id, columnName: column.name }));
        },

        onFiltersChange: (filters: DataTableFilters, column: DataColumn<any>) => {
            dispatch(dataExplorerActions.SET_FILTERS({ id, columnName: column.name, filters }));
        },

        onChangePage: (page: number) => {
            dispatch(dataExplorerActions.SET_PAGE({ id, page }));
        },

        onChangeRowsPerPage: (rowsPerPage: number) => {
            dispatch(dataExplorerActions.SET_ROWS_PER_PAGE({ id, rowsPerPage }));
        },

        onLoadMore: (page: number) => {
            dispatch(dataExplorerActions.SET_PAGE({ id, page }));
        },

        onRowClick,

        onRowDoubleClick,

        onContextMenu,
    });
};

export const DataExplorer = connect(mapStateToProps, mapDispatchToProps)(DataExplorerComponent);

