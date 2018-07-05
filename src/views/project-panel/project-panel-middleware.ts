// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import actions from "../../store/data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID, columns, ProjectPanelFilter } from "./project-panel";
import { groupsService } from "../../services/services";
import { RootState } from "../../store/store";
import { getDataExplorer, DataExplorerState } from "../../store/data-explorer/data-explorer-reducer";
import { resourceToDataItem, ProjectPanelItem } from "./project-panel-item";
import FilterBuilder from "../../common/api/filter-builder";
import { DataColumns } from "../../components/data-table/data-table";
import { ProcessResource } from "../../models/process";
import { CollectionResource } from "../../models/collection";
import OrderBuilder from "../../common/api/order-builder";
import { GroupContentsResource } from "../../services/groups-service/groups-service";

export const projectPanelMiddleware: Middleware = store => next => {
    next(actions.SET_COLUMNS({ id: PROJECT_PANEL_ID, columns }));

    return action => {

        const handleProjectPanelAction = <T extends { id: string }>(handler: (data: T) => void) =>
            (data: T) => {
                next(action);
                if (data.id === PROJECT_PANEL_ID) {
                    handler(data);
                }
            };

        actions.match(action, {
            SET_PAGE: handleProjectPanelAction(() => {
                store.dispatch(actions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            SET_ROWS_PER_PAGE: handleProjectPanelAction(() => {
                store.dispatch(actions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            SET_FILTERS: handleProjectPanelAction(() => {
                store.dispatch(actions.RESET_PAGINATION({ id: PROJECT_PANEL_ID }));
                store.dispatch(actions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            TOGGLE_SORT: handleProjectPanelAction(() => {
                store.dispatch(actions.RESET_PAGINATION({ id: PROJECT_PANEL_ID }));
                store.dispatch(actions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            REQUEST_ITEMS: handleProjectPanelAction(() => {
                const state = store.getState() as RootState;
                const dataExplorer = getDataExplorer(state.dataExplorer, PROJECT_PANEL_ID);
                const columns = dataExplorer.columns as DataColumns<ProjectPanelItem, ProjectPanelFilter>;
                const typeFilters = getColumnFilters(columns, "Type");
                const statusFilters = getColumnFilters(columns, "Status");
                const sortColumn = dataExplorer.columns.find(({ sortDirection }) => Boolean(sortDirection && sortDirection !== "none"));
                const sortDirection = sortColumn && sortColumn.sortDirection === "asc" ? "asc" : "desc";
                if (typeFilters.length > 0) {
                    groupsService
                        .contents(state.projects.currentItemId, {
                            limit: dataExplorer.rowsPerPage,
                            offset: dataExplorer.page * dataExplorer.rowsPerPage,
                            order: sortColumn
                                ? sortColumn.name === "Name"
                                    ? getOrder("name", sortDirection)
                                    : getOrder("createdAt", sortDirection)
                                : OrderBuilder.create(),
                            filters: FilterBuilder
                                .create()
                                .concat(FilterBuilder
                                    .create<CollectionResource>("collections")
                                    .addIsA("uuid", typeFilters.map(f => f.type)))
                                .concat(FilterBuilder
                                    .create<ProcessResource>("containerRequests")
                                    .addIn("state", statusFilters.map(f => f.type)))
                        })
                        .then(response => {
                            store.dispatch(actions.SET_ITEMS({
                                id: PROJECT_PANEL_ID,
                                items: response.items.map(resourceToDataItem),
                                itemsAvailable: response.itemsAvailable,
                                page: Math.floor(response.offset / response.limit),
                                rowsPerPage: response.limit
                            }));
                        });
                } else {
                    store.dispatch(actions.SET_ITEMS({
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

const getOrder = (attribute: "name" | "createdAt", direction: "asc" | "desc") =>
    [
        OrderBuilder.create<GroupContentsResource>("collections"),
        OrderBuilder.create<GroupContentsResource>("container_requests"),
        OrderBuilder.create<GroupContentsResource>("groups")
    ].reduce((acc, b) => acc.concat(direction === "asc"
        ? b.addAsc(attribute)
        : b.addDesc(attribute)),
        OrderBuilder.create());


