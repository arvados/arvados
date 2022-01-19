// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService,
    dataExplorerToListParams,
    getDataExplorerColumnFilters,
    listResultsToDataExplorerItemsMeta
} from 'store/data-explorer/data-explorer-middleware-service';
import { ProjectPanelColumnNames } from "views/project-panel/project-panel";
import { RootState } from "store/store";
import { DataColumns } from "components/data-table/data-table";
import { ServiceRepository } from "services/services";
import { SortDirection } from "components/data-table/data-column";
import { OrderBuilder, OrderDirection } from "services/api/order-builder";
import { FilterBuilder, joinFilters } from "services/api/filter-builder";
import { GroupContentsResource, GroupContentsResourcePrefix } from "services/groups-service/groups-service";
import { updateFavorites } from "store/favorites/favorites-actions";
import { IS_PROJECT_PANEL_TRASHED, projectPanelActions, getProjectPanelCurrentUuid } from 'store/project-panel/project-panel-action';
import { Dispatch, MiddlewareAPI } from "redux";
import { ProjectResource } from "models/project";
import { updateResources } from "store/resources/resources-actions";
import { getProperty } from "store/properties/properties";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { DataExplorer, getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { ListResults } from 'services/common-service/common-service';
import { loadContainers } from 'store/processes/processes-actions';
import { ResourceKind } from 'models/resource';
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { serializeResourceTypeFilters, ProcessStatusFilter } from 'store/resource-type-filters/resource-type-filters';
import { updatePublicFavorites } from 'store/public-favorites/public-favorites-actions';

export class ProjectPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
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
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const response = await this.services.groupsService.contents(projectUuid, getParams(dataExplorer, !!isProjectTrashed));
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                const resourceUuids = response.items.map(item => item.uuid);
                api.dispatch<any>(updateFavorites(resourceUuids));
                api.dispatch<any>(updatePublicFavorites(resourceUuids));
                api.dispatch(updateResources(response.items));
                await api.dispatch<any>(loadMissingProcessesInformation(response.items));
                api.dispatch(setItems(response));
            } catch (e) {
                api.dispatch(projectPanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchProjectContents());
            } finally {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
            }
        }
    }
}

export const loadMissingProcessesInformation = (resources: GroupContentsResource[]) =>
    async (dispatch: Dispatch) => {
        const containerUuids = resources.reduce((uuids, resource) => {
            return resource.kind === ResourceKind.CONTAINER_REQUEST
                ? resource.containerUuid
                    ? [...uuids, resource.containerUuid]
                    : uuids
                : uuids;
        }, []);
        if (containerUuids.length > 0) {
            await dispatch<any>(loadContainers(
                new FilterBuilder().addIn('uuid', containerUuids).getFilters()
            ));
        }
    };

export const setItems = (listResults: ListResults<GroupContentsResource>) =>
    projectPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

export const getParams = (dataExplorer: DataExplorer, isProjectTrashed: boolean) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer),
    includeTrash: isProjectTrashed
});

export const getFilters = (dataExplorer: DataExplorer) => {
    const columns = dataExplorer.columns as DataColumns<string>;
    const typeFilters = serializeResourceTypeFilters(getDataExplorerColumnFilters(columns, ProjectPanelColumnNames.TYPE));
    const statusColumnFilters = getDataExplorerColumnFilters(columns, 'Status');
    const activeStatusFilter = Object.keys(statusColumnFilters).find(
        filterName => statusColumnFilters[filterName].selected
    );

    // TODO: Extract group contents name filter
    const nameFilters = new FilterBuilder()
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROCESS)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
        .getFilters();

    // Filter by container status
    const fb = new FilterBuilder();
    switch (activeStatusFilter) {
        case ProcessStatusFilter.COMPLETED: {
            fb.addEqual('container.state', 'Complete', GroupContentsResourcePrefix.PROCESS);
            fb.addEqual('container.exit_code', '0', GroupContentsResourcePrefix.PROCESS);
            break;
        }
        case ProcessStatusFilter.FAILED: {
            fb.addEqual('container.state', 'Complete', GroupContentsResourcePrefix.PROCESS);
            fb.addDistinct('container.exit_code', '0', GroupContentsResourcePrefix.PROCESS);
            break;
        }
        case ProcessStatusFilter.CANCELLED:
        case ProcessStatusFilter.LOCKED:
        case ProcessStatusFilter.QUEUED:
        case ProcessStatusFilter.RUNNING: {
            fb.addEqual('container.state', activeStatusFilter, GroupContentsResourcePrefix.PROCESS);
            break;
        }
    }
    const statusFilters = fb.getFilters();

    return joinFilters(
        statusFilters,
        typeFilters,
        nameFilters,
    );
};

export const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<ProjectResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        const columnName = sortColumn && sortColumn.name === ProjectPanelColumnNames.NAME ? "name" : "createdAt";
        return order
            .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.COLLECTION)
            .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.PROCESS)
            .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.PROJECT)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

const projectPanelCurrentUuidIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Project panel is not opened.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchProjectContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch project contents.',
        kind: SnackbarKind.ERROR
    });

const projectPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Project panel is not ready.',
        kind: SnackbarKind.ERROR
    });
