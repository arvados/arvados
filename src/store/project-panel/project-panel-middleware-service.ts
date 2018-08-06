// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { ProjectPanelColumnNames, ProjectPanelFilter } from "../../views/project-panel/project-panel";
import { RootState } from "../store";
import { DataColumns } from "../../components/data-table/data-table";
import { ServiceRepository } from "../../services/services";
import { ProjectPanelItem, resourceToDataItem } from "../../views/project-panel/project-panel-item";
import { SortDirection } from "../../components/data-table/data-column";
import { OrderBuilder } from "../../common/api/order-builder";
import { FilterBuilder } from "../../common/api/filter-builder";
import { ProcessResource } from "../../models/process";
import { GroupContentsResourcePrefix, GroupContentsResource } from "../../services/groups-service/groups-service";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";
import { projectPanelActions } from "./project-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";

export class ProjectPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = state.dataExplorer[this.getId()];
        const columns = dataExplorer.columns as DataColumns<ProjectPanelItem, ProjectPanelFilter>;
        const typeFilters = getColumnFilters(columns, ProjectPanelColumnNames.TYPE);
        const statusFilters = getColumnFilters(columns, ProjectPanelColumnNames.STATUS);
        const sortColumn = dataExplorer.columns.find(({ sortDirection }) => Boolean(sortDirection && sortDirection !== "none"));
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC ? SortDirection.ASC : SortDirection.DESC;
        if (typeFilters.length > 0) {
            this.services.groupsService
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
                            .create(GroupContentsResourcePrefix.PROCESS)
                            .addIn("state", statusFilters.map(f => f.type)))
                        .concat(getSearchFilter(dataExplorer.searchValue))
                })
                .then(response => {
                    api.dispatch(projectPanelActions.SET_ITEMS({
                        items: response.items.map(resourceToDataItem),
                        itemsAvailable: response.itemsAvailable,
                        page: Math.floor(response.offset / response.limit),
                        rowsPerPage: response.limit
                    }));
                    api.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
                });
        } else {
            api.dispatch(projectPanelActions.SET_ITEMS({
                items: [],
                itemsAvailable: 0,
                page: 0,
                rowsPerPage: dataExplorer.rowsPerPage
            }));
        }
    }
}

const getColumnFilters = (columns: DataColumns<ProjectPanelItem, ProjectPanelFilter>, columnName: string) => {
    const column = columns.find(c => c.name === columnName);
    return column && column.filters ? column.filters.filter(f => f.selected) : [];
};

const getOrder = (attribute: "name" | "createdAt", direction: SortDirection) =>
    [
        OrderBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.COLLECTION),
        OrderBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.PROCESS),
        OrderBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.PROJECT)
    ].reduce((acc, b) =>
        acc.concat(direction === SortDirection.ASC
            ? b.addAsc(attribute)
            : b.addDesc(attribute)), OrderBuilder.create());

const getSearchFilter = (searchValue: string) =>
    searchValue
        ? [
            FilterBuilder.create(GroupContentsResourcePrefix.COLLECTION),
            FilterBuilder.create(GroupContentsResourcePrefix.PROCESS),
            FilterBuilder.create(GroupContentsResourcePrefix.PROJECT)]
            .reduce((acc, b) =>
                acc.concat(b.addILike("name", searchValue)), FilterBuilder.create())
        : FilterBuilder.create();
