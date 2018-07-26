// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { PROJECT_PANEL_ID, columns, ProjectPanelColumnNames, ProjectPanelFilter } from "../../views/project-panel/project-panel";
import { getDataExplorer } from "../data-explorer/data-explorer-reducer";
import { RootState } from "../store";
import { DataColumns } from "../../components/data-table/data-table";
import { groupsService } from "../../services/services";
import { ProjectPanelItem, resourceToDataItem } from "../../views/project-panel/project-panel-item";
import { SortDirection } from "../../components/data-table/data-column";
import { OrderBuilder } from "../../common/api/order-builder";
import { FilterBuilder } from "../../common/api/filter-builder";
import { ProcessResource } from "../../models/process";
import { GroupContentsResourcePrefix, GroupContentsResource } from "../../services/groups-service/groups-service";
import { dataExplorerActions } from "../data-explorer/data-explorer-action";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";

export class ProjectPanelMiddlewareService extends DataExplorerMiddlewareService {

    private static instance: ProjectPanelMiddlewareService;

    static getInstance() {
        return ProjectPanelMiddlewareService.instance
            ? ProjectPanelMiddlewareService.instance
            : new ProjectPanelMiddlewareService();
    }

    private constructor() {
        super();
    }

    get Id() {
        return PROJECT_PANEL_ID;
    }

    get Columns() {
        return columns;
    }

    requestItems() {
        const state = this.api.getState() as RootState;
        const dataExplorer = getDataExplorer(state.dataExplorer, PROJECT_PANEL_ID);
        const columns = dataExplorer.columns as DataColumns<ProjectPanelItem, ProjectPanelFilter>;
        const typeFilters = getColumnFilters(columns, ProjectPanelColumnNames.TYPE);
        const statusFilters = getColumnFilters(columns, ProjectPanelColumnNames.STATUS);
        const sortColumn = dataExplorer.columns.find(({ sortDirection }) => Boolean(sortDirection && sortDirection !== "none"));
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC ? SortDirection.ASC : SortDirection.DESC;
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
                            .create<ProcessResource>(GroupContentsResourcePrefix.PROCESS)
                            .addIn("state", statusFilters.map(f => f.type)))
                        .concat(getSearchFilter(dataExplorer.searchValue))
                })
                .then(response => {
                    this.api.dispatch(dataExplorerActions.SET_ITEMS({
                        id: PROJECT_PANEL_ID,
                        items: response.items.map(resourceToDataItem),
                        itemsAvailable: response.itemsAvailable,
                        page: Math.floor(response.offset / response.limit),
                        rowsPerPage: response.limit
                    }));
                    this.api.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
                });
        } else {
            this.api.dispatch(dataExplorerActions.SET_ITEMS({
                id: PROJECT_PANEL_ID,
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
            FilterBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.COLLECTION),
            FilterBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.PROCESS),
            FilterBuilder.create<GroupContentsResource>(GroupContentsResourcePrefix.PROJECT)]
            .reduce((acc, b) =>
                acc.concat(b.addILike("name", searchValue)), FilterBuilder.create())
        : FilterBuilder.create();