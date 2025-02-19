// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService,
    dataExplorerToListParams,
    getDataExplorerColumnFilters,
    listResultsToDataExplorerItemsMeta,
} from "store/data-explorer/data-explorer-middleware-service";
import { ProjectPanelDataColumnNames } from "views/project-panel/project-panel-data";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { OrderBuilder, OrderDirection } from "services/api/order-builder";
import { FilterBuilder, joinFilters } from "services/api/filter-builder";
import { ContentsArguments, GroupContentsResource, GroupContentsResourcePrefix } from "services/groups-service/groups-service";
import { updateFavorites } from "store/favorites/favorites-actions";
import { IS_PROJECT_PANEL_TRASHED, getProjectPanelCurrentUuid } from "store/project-panel/project-panel";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { Dispatch, MiddlewareAPI } from "redux";
import { ProjectResource } from "models/project";
import { updateResources } from "store/resources/resources-actions";
import { getProperty } from "store/properties/properties";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { DataExplorer, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { ListResults } from "services/common-service/common-service";
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { buildProcessStatusFilters, serializeDataResourceTypeFilters } from "store/resource-type-filters/resource-type-filters";
import { updatePublicFavorites } from "store/public-favorites/public-favorites-actions";
import { selectedFieldsOfGroup } from "models/group";
import { defaultCollectionSelectedFields } from "models/collection";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { removeDisabledButton } from "store/multiselect/multiselect-actions";
import { couldNotFetchItemsAvailable } from "store/data-explorer/data-explorer-action";

export class ProjectPanelDataMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const projectUuid = getProjectPanelCurrentUuid(state);
        const isProjectTrashed = getProperty<string>(IS_PROJECT_PANEL_TRASHED)(state.properties);
        if (!projectUuid) {
            api.dispatch(projectPanelCurrentUuidIsNotSet());
        } else if (!dataExplorer) {
            api.dispatch(projectPanelDataExplorerIsNotSet());
        } else {
            try {
                api.dispatch<any>(projectPanelDataActions.SET_IS_NOT_FOUND({ isNotFound: false }));
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

                // Get items
                const response = await this.services.groupsService.contents(projectUuid, getParams(dataExplorer, !!isProjectTrashed));
                const resourceUuids = [...response.items.map(item => item.uuid), projectUuid];
                api.dispatch<any>(updateFavorites(resourceUuids));
                api.dispatch<any>(updatePublicFavorites(resourceUuids));
                api.dispatch(updateResources(response.items));
                api.dispatch(setItems(response));
            } catch (e) {
                api.dispatch(
                    projectPanelDataActions.SET_ITEMS({
                        items: [],
                        itemsAvailable: 0,
                        page: 0,
                        rowsPerPage: dataExplorer.rowsPerPage,
                    })
                );
                if (e.status === 404) {
                    api.dispatch<any>(projectPanelDataActions.SET_IS_NOT_FOUND({ isNotFound: true}));
                }
                else {
                    api.dispatch(couldNotFetchProjectContents());
                }
            } finally {
                if (!background) {
                    api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
                    api.dispatch<any>(removeDisabledButton(ContextMenuActionNames.MOVE_TO_TRASH))
                }
            }
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const projectUuid = getProjectPanelCurrentUuid(state);
        const isProjectTrashed = getProperty<string>(IS_PROJECT_PANEL_TRASHED)(state.properties);

        if (criteriaChanged && projectUuid) {
            // Get itemsAvailable
            return this.services.groupsService.contents(projectUuid, getCountParams(dataExplorer, !!isProjectTrashed))
                .then((results: ListResults<GroupContentsResource>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(projectPanelDataActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                });
        }
    }
}

export const setItems = (listResults: ListResults<GroupContentsResource>) =>
    projectPanelDataActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

export const getParams = (dataExplorer: DataExplorer, isProjectTrashed: boolean): ContentsArguments => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer),
    includeTrash: isProjectTrashed,
    select: selectedFieldsOfGroup.concat(defaultCollectionSelectedFields),
    count: 'none',
});

const getCountParams = (dataExplorer: DataExplorer, isProjectTrashed: boolean): ContentsArguments => ({
    filters: getFilters(dataExplorer),
    includeTrash: isProjectTrashed,
    limit: 0,
    count: 'exact',
});

export const getFilters = (dataExplorer: DataExplorer) => {
    const columns = dataExplorer.columns as DataColumns<string, ProjectResource>;
    const typeFilters = serializeDataResourceTypeFilters(getDataExplorerColumnFilters(columns, ProjectPanelDataColumnNames.TYPE));
    const statusColumnFilters = getDataExplorerColumnFilters(columns, "Status");
    const activeStatusFilter = Object.keys(statusColumnFilters).find(filterName => statusColumnFilters[filterName].selected);

    // TODO: Extract group contents name filter
    const nameFilters = new FilterBuilder()
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
        .getFilters();

    // Filter by container status
    const statusFilters = buildProcessStatusFilters(new FilterBuilder(), activeStatusFilter || "", GroupContentsResourcePrefix.PROCESS).getFilters();

    return joinFilters(statusFilters, typeFilters, nameFilters);
};

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn<ProjectResource>(dataExplorer);
    const order = new OrderBuilder<ProjectResource>();
    if (sortColumn && sortColumn.sort) {
        const sortDirection = sortColumn.sort.direction === SortDirection.ASC ? OrderDirection.ASC : OrderDirection.DESC;

        // Use createdAt as a secondary sort column so we break ties consistently.
        return order
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.COLLECTION)
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.PROJECT)
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.WORKFLOW)
            .addOrder(OrderDirection.DESC, "createdAt")
            .getOrder();
    } else {
        return order.getOrder();
    }
};

const projectPanelCurrentUuidIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: "Project panel is not opened.",
        kind: SnackbarKind.ERROR,
    });

const couldNotFetchProjectContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: "Could not fetch project contents.",
        kind: SnackbarKind.ERROR,
    });

const projectPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: "Project panel is not ready.",
        kind: SnackbarKind.ERROR,
    });
