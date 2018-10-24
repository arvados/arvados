// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService,
    dataExplorerToListParams,
    getDataExplorerColumnFilters,
    listResultsToDataExplorerItemsMeta
} from '../data-explorer/data-explorer-middleware-service';
import { ProjectPanelColumnNames, ProjectPanelFilter } from "~/views/project-panel/project-panel";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { ServiceRepository } from "~/services/services";
import { SortDirection } from "~/components/data-table/data-column";
import { OrderBuilder, OrderDirection } from "~/services/api/order-builder";
import { FilterBuilder } from "~/services/api/filter-builder";
import { GroupContentsResource, GroupContentsResourcePrefix } from "~/services/groups-service/groups-service";
import { updateFavorites } from "../favorites/favorites-actions";
import { PROJECT_PANEL_CURRENT_UUID, projectPanelActions } from './project-panel-action';
import { Dispatch, MiddlewareAPI } from "redux";
import { ProjectResource } from "~/models/project";
import { updateResources } from "~/store/resources/resources-actions";
import { getProperty } from "~/store/properties/properties";
import { snackbarActions, SnackbarKind } from '../snackbar/snackbar-actions';
import { progressIndicatorActions } from '~/store/progress-indicator/progress-indicator-actions.ts';
import { DataExplorer, getDataExplorer } from '../data-explorer/data-explorer-reducer';
import { ListResults } from '~/services/common-service/common-resource-service';
import { loadContainers } from '../processes/processes-actions';
import { ResourceKind } from '~/models/resource';
import { getResource } from "~/store/resources/resources";
import { CollectionResource } from "~/models/collection";
import { resourcesDataActions } from "~/store/resources-data/resources-data-actions";

export class ProjectPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const projectUuid = getProperty<string>(PROJECT_PANEL_CURRENT_UUID)(state.properties);
        if (!projectUuid) {
            api.dispatch(projectPanelCurrentUuidIsNotSet());
        } else if (!dataExplorer) {
            api.dispatch(projectPanelDataExplorerIsNotSet());
        } else {
            try {
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const response = await this.services.groupsService.contents(projectUuid, getParams(dataExplorer));
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                const resourceUuids = response.items.map(item => item.uuid);
                api.dispatch<any>(updateFavorites(resourceUuids));
                api.dispatch(updateResources(response.items));
                api.dispatch<any>(updateResourceData(resourceUuids));
                await api.dispatch<any>(loadMissingProcessesInformation(response.items));
                api.dispatch(setItems(response));
            } catch (e) {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(projectPanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchProjectContents());
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

export const updateResourceData = (resourceUuids: string[]) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        resourceUuids.map(async uuid => {
            const resource = getResource<CollectionResource>(uuid)(getState().resources);
            if (resource && resource.kind === ResourceKind.COLLECTION) {
                const files = await services.collectionService.files(uuid);
                if (files) {
                    dispatch(resourcesDataActions.SET_FILES({ uuid, files }));
                }
            }
        });
    };

export const setItems = (listResults: ListResults<GroupContentsResource>) =>
    projectPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

export const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer),
});

export const getFilters = (dataExplorer: DataExplorer) => {
    const columns = dataExplorer.columns as DataColumns<string, ProjectPanelFilter>;
    const typeFilters = getDataExplorerColumnFilters(columns, ProjectPanelColumnNames.TYPE);
    const statusFilters = getDataExplorerColumnFilters(columns, ProjectPanelColumnNames.STATUS);
    return new FilterBuilder()
        .addIsA("uuid", typeFilters.map(f => f.type))
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROCESS)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
        .getFilters();
};

export const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = dataExplorer.columns.find(c => c.sortDirection !== SortDirection.NONE);
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
        message: 'Project panel is not opened.'
    });

const couldNotFetchProjectContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch project contents.',
        kind: SnackbarKind.ERROR
    });

const projectPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Project panel is not ready.'
    });
