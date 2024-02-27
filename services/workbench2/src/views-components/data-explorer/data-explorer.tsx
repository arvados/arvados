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
import { DataColumns, TCheckedList } from "components/data-table/data-table";
import { DataTableFilters } from "components/data-table-filters/data-table-filters-tree";
import { LAST_REFRESH_TIMESTAMP } from "components/refresh-button/refresh-button";
import { toggleMSToolbar, setCheckedListOnStore } from "store/multiselect/multiselect-actions";

interface Props {
    id: string;
    onRowClick: (item: any) => void;
    onContextMenu?: (event: React.MouseEvent<HTMLElement>, item: any, isAdmin?: boolean) => void;
    onRowDoubleClick: (item: any) => void;
    extractKey?: (item: any) => React.Key;
    working?: boolean;
}

const mapStateToProps = ({ progressIndicator, dataExplorer, router, multiselect, detailsPanel, properties}: RootState, { id }: Props) => {
    const working = !!progressIndicator.some(p => p.id === id && p.working);
    const dataExplorerState = getDataExplorer(dataExplorer, id);
    const currentRoute = router.location ? router.location.pathname : "";
    const currentRefresh = localStorage.getItem(LAST_REFRESH_TIMESTAMP) || "";
    const isDetailsResourceChecked = multiselect.checkedList[detailsPanel.resourceUuid]
    const isOnlyOneSelected = Object.values(multiselect.checkedList).filter(x => x === true).length === 1;
    const currentItemUuid =
        currentRoute === '/workflows' ? properties.workflowPanelDetailsUuid : isDetailsResourceChecked && isOnlyOneSelected ? detailsPanel.resourceUuid : multiselect.selectedUuid;
    const isMSToolbarVisible = multiselect.isVisible;
    return {
        ...dataExplorerState,
        currentRefresh: currentRefresh,
        currentRoute: currentRoute,
        paperKey: currentRoute,
        currentItemUuid,
        isMSToolbarVisible,
        checkedList: multiselect.checkedList,
        working,
        isNotFound: dataExplorerState.isNotFound,
    };
};

const mapDispatchToProps = () => {
    return (dispatch: Dispatch, { id, onRowClick, onRowDoubleClick, onContextMenu }: Props) => ({
        onSetColumns: (columns: DataColumns<any, any>) => {
            dispatch(dataExplorerActions.SET_COLUMNS({ id, columns }));
        },

        onSearch: (searchValue: string) => {
            dispatch(dataExplorerActions.SET_EXPLORER_SEARCH_VALUE({ id, searchValue }));
        },

        onColumnToggle: (column: DataColumn<any, any>) => {
            dispatch(dataExplorerActions.TOGGLE_COLUMN({ id, columnName: column.name }));
        },

        onSortToggle: (column: DataColumn<any, any>) => {
            dispatch(dataExplorerActions.TOGGLE_SORT({ id, columnName: column.name }));
        },

        onFiltersChange: (filters: DataTableFilters, column: DataColumn<any, any>) => {
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

        toggleMSToolbar: (isVisible: boolean) => {
            dispatch<any>(toggleMSToolbar(isVisible));
        },

        setCheckedListOnStore: (checkedList: TCheckedList) => {
            dispatch<any>(setCheckedListOnStore(checkedList));
        },

        onRowClick,

        onRowDoubleClick,

        onContextMenu,
    });
};

export const DataExplorer = connect(mapStateToProps, mapDispatchToProps)(DataExplorerComponent);
