// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "../../store/store";
import DataExplorer from "../../components/data-explorer/data-explorer";
import { getDataExplorer } from "../../store/data-explorer/data-explorer-reducer";
import { Dispatch } from "redux";
import actions from "../../store/data-explorer/data-explorer-action";
import { DataColumn } from "../../components/data-table/data-column";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";
import { ContextMenuAction, ContextMenuActionGroup } from "../../components/context-menu/context-menu";

interface Props {
    id: string;
    onRowClick: (item: any) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => void;
}

const mapStateToProps = (state: RootState, { id }: Props) =>
    getDataExplorer(state.dataExplorer, id);

const mapDispatchToProps = (dispatch: Dispatch, { id, onRowClick, onContextMenu }: Props) => ({
    onSearch: (searchValue: string) => {
        dispatch(actions.SET_SEARCH_VALUE({ id, searchValue }));
    },

    onColumnToggle: (column: DataColumn<any>) => {
        dispatch(actions.TOGGLE_COLUMN({ id, columnName: column.name }));
    },

    onSortToggle: (column: DataColumn<any>) => {
        dispatch(actions.TOGGLE_SORT({ id, columnName: column.name }));
    },

    onFiltersChange: (filters: DataTableFilterItem[], column: DataColumn<any>) => {
        dispatch(actions.SET_FILTERS({ id, columnName: column.name, filters }));
    },

    onChangePage: (page: number) => {
        dispatch(actions.SET_PAGE({ id, page }));
    },

    onChangeRowsPerPage: (rowsPerPage: number) => {
        dispatch(actions.SET_ROWS_PER_PAGE({ id, rowsPerPage }));
    },

    onRowClick,

    onContextMenu
});

export default connect(mapStateToProps, mapDispatchToProps)(DataExplorer);

