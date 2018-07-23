// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import { dataExplorerActions } from "../data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID, columns, ProjectPanelFilter, ProjectPanelColumnNames } from "../../views/project-panel/project-panel";
import { groupsService } from "../../services/services";
import { RootState } from "../store";
import { getDataExplorer } from "../data-explorer/data-explorer-reducer";
import { resourceToDataItem, ProjectPanelItem } from "../../views/project-panel/project-panel-item";
import { FilterBuilder } from "../../common/api/filter-builder";
import { DataColumns } from "../../components/data-table/data-table";
import { ProcessResource } from "../../models/process";
import { OrderBuilder } from "../../common/api/order-builder";
import { GroupContentsResource, GroupContentsResourcePrefix } from "../../services/groups-service/groups-service";
import { SortDirection } from "../../components/data-table/data-column";

export const projectPanelMiddleware: Middleware = store => next => {
    next(dataExplorerActions.SET_COLUMNS({ id: PROJECT_PANEL_ID, columns }));

    return action => {

        const handleProjectPanelAction = <T extends { id: string }>(handler: (data: T) => void) =>
            (data: T) => {
                next(action);
                if (data.id === PROJECT_PANEL_ID) {
                    handler(data);
                }
            };

        dataExplorerActions.match(action, {
            SET_PAGE: handleProjectPanelAction(() => {
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            SET_ROWS_PER_PAGE: handleProjectPanelAction(() => {
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            SET_FILTERS: handleProjectPanelAction(() => {
                store.dispatch(dataExplorerActions.RESET_PAGINATION({ id: PROJECT_PANEL_ID }));
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            TOGGLE_SORT: handleProjectPanelAction(() => {
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            SET_SEARCH_VALUE: handleProjectPanelAction(() => {
                store.dispatch(dataExplorerActions.RESET_PAGINATION({ id: PROJECT_PANEL_ID }));
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            REQUEST_ITEMS: handleProjectPanelAction(() => {
                const state = store.getState() as RootState;
                const dataExplorer = getDataExplorer(state.dataExplorer, PROJECT_PANEL_ID);
                const columns = dataExplorer.columns as DataColumns<ProjectPanelItem, ProjectPanelFilter>;
                const typeFilters = getColumnFilters(columns, ProjectPanelColumnNames.TYPE);
                const statusFilters = getColumnFilters(columns, ProjectPanelColumnNames.STATUS);
                const sortColumn = dataExplorer.columns.find(({ sortDirection }) => Boolean(sortDirection && sortDirection !== "none"));
                const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.Asc ? SortDirection.Asc : SortDirection.Desc;
                if (typeFilters.length > 0) {
                    groupsService
                        .contents(state.projects.currentItemId, {
                            limit: dataExplorer.rowsPerPage,
                            offset: dataExplorer.page * dataExplorer.rowsPerPage,
                            order: sortColumn
                                ? sortColumn.name === ProjectPanelColumnNames.NAME
                                    ? getOrder("name", sortDirection)
                                    : getOrder("createdAt", sortDirection)
                                : OrderBuilder.create(),
                            filters: FilterBuilder
                                .create()
                                .concat(FilterBuilder
                                    .create()
                                    .addIsA("uuid", typeFilters.map(f => f.type)))
                                .concat(FilterBuilder
                                    .create<ProcessResource>(GroupContentsResourcePrefix.Process)
                                    .addIn("state", statusFilters.map(f => f.type)))
                                .concat(getSearchFilter(dataExplorer.searchValue))
                        })
                        .then(response => {
                            store.dispatch(dataExplorerActions.SET_ITEMS({
                                id: PROJECT_PANEL_ID,
                                items: response.items.map(resourceToDataItem),
                                itemsAvailable: response.itemsAvailable,
                                page: Math.floor(response.offset / response.limit),
                                rowsPerPage: response.limit
                            }));
                        });
                } else {
                    store.dispatch(dataExplorerActions.SET_ITEMS({
                        id: PROJECT_PANEL_ID,
                        items: [],
                        itemsAvailable: 0,
                        page: 0,
                        rowsPerPage: dataExplorer.rowsPerPage
                    }));
                }
            }),
            default: () => next(action)
        });
    };
};

const getColumnFilters = (columns: DataColumns<ProjectPanelItem, ProjectPanelFilter>, columnName: string) => {
    const column = columns.find(c => c.name === columnName);
    return column && column.filters ? column.filters.filter(f => f.selected) : [];
};

const getOrder = (attribute: "name" | "createdAt", direction: SortDirection) =>
    [
        OrderBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.Collection),
        OrderBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.Process),
        OrderBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.Project)
    ].reduce((acc, b) =>
        acc.concat(direction === SortDirection.Asc
            ? b.addAsc(attribute)
            : b.addDesc(attribute)), OrderBuilder.create());

const getSearchFilter = (searchValue: string) =>
    searchValue
        ? [
            FilterBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.Collection),
            FilterBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.Process),
            FilterBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.Project)]
            .reduce((acc, b) =>
                acc.concat(b.addILike("name", searchValue)), FilterBuilder.create())
        : FilterBuilder.create();


