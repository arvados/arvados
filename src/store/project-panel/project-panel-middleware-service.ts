// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { ProjectPanelColumnNames, ProjectPanelFilter } from "~/views/project-panel/project-panel";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { ServiceRepository } from "~/services/services";
import { SortDirection } from "~/components/data-table/data-column";
import { OrderBuilder, OrderDirection } from "~/common/api/order-builder";
import { FilterBuilder } from "~/common/api/filter-builder";
import { GroupContentsResourcePrefix } from "~/services/groups-service/groups-service";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";
import { projectPanelActions } from "./project-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { ProjectResource } from "~/models/project";
import { resourcesActions } from "~/store/resources/resources-actions";

export class ProjectPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = state.dataExplorer[this.getId()];
        const columns = dataExplorer.columns as DataColumns<string, ProjectPanelFilter>;
        const typeFilters = this.getColumnFilters(columns, ProjectPanelColumnNames.TYPE);
        const statusFilters = this.getColumnFilters(columns, ProjectPanelColumnNames.STATUS);
        const sortColumn = dataExplorer.columns.find(c => c.sortDirection !== SortDirection.NONE);

        const order = new OrderBuilder<ProjectResource>();

        if (sortColumn) {
            const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
                ? OrderDirection.ASC
                : OrderDirection.DESC;

            const columnName = sortColumn && sortColumn.name === ProjectPanelColumnNames.NAME ? "name" : "createdAt";
            order
                .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.COLLECTION)
                .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.PROCESS)
                .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.PROJECT);
        }

        this.services.groupsService
            .contents(state.projects.currentItemId, {
                limit: dataExplorer.rowsPerPage,
                offset: dataExplorer.page * dataExplorer.rowsPerPage,
                order: order.getOrder(),
                filters: new FilterBuilder()
                    .addIsA("uuid", typeFilters.map(f => f.type))
                    .addIn("state", statusFilters.map(f => f.type), GroupContentsResourcePrefix.PROCESS)
                    .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
                    .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROCESS)
                    .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
                    .getFilters()
            })
            .then(response => {
                api.dispatch(resourcesActions.SET_RESOURCES(response.items));
                api.dispatch(projectPanelActions.SET_ITEMS({
                    items: response.items.map(resource => resource.uuid),
                    itemsAvailable: response.itemsAvailable,
                    page: Math.floor(response.offset / response.limit),
                    rowsPerPage: response.limit
                }));
                api.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
            })
            .catch(() => {
                api.dispatch(projectPanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
            });
    }
}
