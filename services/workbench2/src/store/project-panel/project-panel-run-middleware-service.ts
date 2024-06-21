// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService,
    dataExplorerToListParams,
    getDataExplorerColumnFilters,
    listResultsToDataExplorerItemsMeta,
} from "store/data-explorer/data-explorer-middleware-service";
import { ProjectPanelRunColumnNames } from "views/project-panel/project-panel-run";
import { RootState } from "store/store";
import { DataColumns } from "components/data-table/data-table";
import { ServiceRepository } from "services/services";
import { SortDirection } from "components/data-table/data-column";
import { OrderBuilder, OrderDirection } from "services/api/order-builder";
import { FilterBuilder, joinFilters } from "services/api/filter-builder";
import { GroupContentsResource, GroupContentsResourcePrefix } from "services/groups-service/groups-service";
import { updateFavorites } from "store/favorites/favorites-actions";
import { IS_PROJECT_PANEL_TRASHED, getProjectPanelCurrentUuid } from "store/project-panel/project-panel-action";
import { projectPanelRunActions } from "store/project-panel/project-panel-action-bind";
import { Dispatch, MiddlewareAPI } from "redux";
import { ProjectResource } from "models/project";
import { updateResources } from "store/resources/resources-actions";
import { getProperty } from "store/properties/properties";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { DataExplorer, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { ListResults } from "services/common-service/common-service";
import { loadContainers } from "store/processes/processes-actions";
import { ResourceKind } from "models/resource";
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { buildProcessStatusFilters, serializeProcessTypeGroupContentsFilters } from "store/resource-type-filters/resource-type-filters";
import { updatePublicFavorites } from "store/public-favorites/public-favorites-actions";
import { containerRequestFieldsNoMounts } from "models/container-request";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { removeDisabledButton } from "store/multiselect/multiselect-actions";
import { dataExplorerActions } from "store/data-explorer/data-explorer-action";

export class ProjectPanelRunMiddlewareService extends DataExplorerMiddlewareService {
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
                api.dispatch<any>(dataExplorerActions.SET_IS_NOT_FOUND({ id: this.id, isNotFound: false }));
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }
                const containerRequests = await this.services.groupsService.contents(projectUuid, getParams(dataExplorer, projectUuid, !!isProjectTrashed));
                const resourceUuids = containerRequests.items.map(item => item.uuid);
                api.dispatch<any>(updateFavorites(resourceUuids));
                api.dispatch<any>(updatePublicFavorites(resourceUuids));
                api.dispatch(updateResources(containerRequests.items));
                await api.dispatch<any>(loadMissingProcessesInformation(containerRequests.items));
                api.dispatch(setItems(containerRequests));
            } catch (e) {
                api.dispatch(
                    projectPanelRunActions.SET_ITEMS({
                        items: [],
                        itemsAvailable: 0,
                        page: 0,
                        rowsPerPage: dataExplorer.rowsPerPage,
                    })
                );
                if (e.status === 404) {
                    api.dispatch<any>(dataExplorerActions.SET_IS_NOT_FOUND({ id: this.id, isNotFound: true}));
                }
                else {
                    api.dispatch(couldNotFetchProjectContents());
                }
            } finally {
                if (!background) {
                    api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                    api.dispatch<any>(removeDisabledButton(ContextMenuActionNames.MOVE_TO_TRASH))
                }
            }
        }
    }
}

export const loadMissingProcessesInformation = (resources: GroupContentsResource[]) => async (dispatch: Dispatch) => {
    const containerUuids = resources.reduce((uuids, resource) => {
        return resource.kind === ResourceKind.CONTAINER_REQUEST && resource.containerUuid && !uuids.includes(resource.containerUuid)
            ? [...uuids, resource.containerUuid]
            : uuids;
    }, [] as string[]);
    if (containerUuids.length > 0) {
        await dispatch<any>(loadContainers(containerUuids, false));
    }
};

export const setItems = (listResults: ListResults<GroupContentsResource>) =>
    projectPanelRunActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

export const getParams = (dataExplorer: DataExplorer, projectUuid: string, isProjectTrashed: boolean) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer, projectUuid),
    includeTrash: isProjectTrashed,
    select: containerRequestFieldsNoMounts,
});

export const getFilters = (dataExplorer: DataExplorer, projectUuid: string) => {
    const columns = dataExplorer.columns as DataColumns<string, ProjectResource>;
    const typeFilters = serializeProcessTypeGroupContentsFilters(getDataExplorerColumnFilters(columns, ProjectPanelRunColumnNames.TYPE));
    const statusColumnFilters = getDataExplorerColumnFilters(columns, ProjectPanelRunColumnNames.STATUS);
    const activeStatusFilter = Object.keys(statusColumnFilters).find(filterName => statusColumnFilters[filterName].selected);

    // TODO: Extract group contents name filter
    const nameFilters = new FilterBuilder()
        .addEqual('owner_uuid', projectUuid)
        .addILike("name", dataExplorer.searchValue)
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
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.PROCESS)
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
