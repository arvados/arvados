// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { ServiceRepository } from "~/services/services";
import { SortDirection } from "~/components/data-table/data-column";
import { FilterBuilder } from "~/common/api/filter-builder";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";
import { trashPanelActions } from "./trash-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { OrderBuilder, OrderDirection } from "~/common/api/order-builder";
import { GroupContentsResourcePrefix } from "~/services/groups-service/groups-service";
import { resourceToDataItem, TrashPanelItem } from "~/views/trash-panel/trash-panel-item";
import { TrashPanelColumnNames, TrashPanelFilter } from "~/views/trash-panel/trash-panel";
import { ProjectResource } from "~/models/project";
import { ProjectPanelColumnNames } from "~/views/project-panel/project-panel";

export class TrashPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = api.getState().dataExplorer[this.getId()];
        const columns = dataExplorer.columns as DataColumns<TrashPanelItem, TrashPanelFilter>;
        const sortColumn = dataExplorer.columns.find(c => c.sortDirection !== SortDirection.NONE);
        const typeFilters = this.getColumnFilters(columns, TrashPanelColumnNames.TYPE);

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

        const userUuid = this.services.authService.getUuid()!;

        this.services.trashService
            .contents(userUuid, {
                limit: dataExplorer.rowsPerPage,
                offset: dataExplorer.page * dataExplorer.rowsPerPage,
                order: order.getOrder(),
                filters: new FilterBuilder()
                    .addIsA("uuid", typeFilters.map(f => f.type))
                    .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
                    .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROCESS)
                    .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
                    .getFilters(),
                recursive: true,
                includeTrash: true
            })
            .then(response => {
                api.dispatch(trashPanelActions.SET_ITEMS({
                    items: response.items.map(resourceToDataItem).filter(it => it.isTrashed),
                    itemsAvailable: response.itemsAvailable,
                    page: Math.floor(response.offset / response.limit),
                    rowsPerPage: response.limit
                }));
                api.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
            })
            .catch(() => {
                api.dispatch(trashPanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
            });
    }
}
