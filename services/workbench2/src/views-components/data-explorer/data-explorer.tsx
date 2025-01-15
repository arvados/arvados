// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "store/store";
import { DataExplorer as DataExplorerComponent } from "components/data-explorer/data-explorer";
import { getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { Dispatch } from "redux";
import { dataExplorerActions } from "store/data-explorer/data-explorer-action";
import { DataColumn, DataColumns, } from "components/data-table/data-column";
import { TCheckedList } from "components/data-table/data-table";
import { DataTableFilters } from "components/data-table-filters/data-table-filters";
import { toggleMSToolbar, setCheckedListOnStore } from "store/multiselect/multiselect-actions";
import { setSelectedResourceUuid } from "store/selected-resource/selected-resource-actions";
import { usesDetailsCard } from "components/multiselect-toolbar/MultiselectToolbar";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";

interface Props {
    id: string;
    onRowClick: (item: any) => void;
    onContextMenu?: (event: React.MouseEvent<HTMLElement>, item: any, isAdmin?: boolean) => void;
    onRowDoubleClick: (item: any) => void;
    extractKey?: (item: any) => React.Key;
    working?: boolean;
}

const mapStateToProps = ({ progressIndicator, dataExplorer, router, multiselect, selectedResourceUuid, properties, searchBar, detailsPanel}: RootState, { id }: Props) => {
    const working = !!progressIndicator.some(p => p.working);
    const dataExplorerState = getDataExplorer(dataExplorer, id);
    const currentRoute = router.location ? router.location.pathname : "";
    const isMSToolbarVisible = multiselect.isVisible;
    return {
        ...dataExplorerState,
        path: currentRoute,
        currentRouteUuid: properties.currentRouteUuid,
        isMSToolbarVisible,
        selectedResourceUuid,
        checkedList: multiselect.checkedList,
        working,
        searchBarValue: searchBar.searchValue,
        detailsPanelResourceUuid: detailsPanel.resourceUuid,
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

        onPageChange: (page: number) => {
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

        setSelectedUuid: (uuid: string | null) => {
            dispatch<any>(setSelectedResourceUuid(uuid));
        },

        loadDetailsPanel: (uuid: string) => {
            dispatch<any>(loadDetailsPanel(uuid || ''));
        },

        onRowClick,

        onRowDoubleClick,

        onContextMenu,

        usesDetailsCard,
    });
};

export const DataExplorer = connect(mapStateToProps, mapDispatchToProps)(DataExplorerComponent);
