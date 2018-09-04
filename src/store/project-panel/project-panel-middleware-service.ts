// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService, getDataExplorerColumnFilters, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from '../data-explorer/data-explorer-middleware-service';
import { ProjectPanelColumnNames, ProjectPanelFilter } from "~/views/project-panel/project-panel";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { ServiceRepository } from "~/services/services";
import { SortDirection } from "~/components/data-table/data-column";
import { OrderBuilder, OrderDirection } from "~/services/api/order-builder";
import { FilterBuilder } from "~/services/api/filter-builder";
import { GroupContentsResourcePrefix, GroupContentsResource } from "~/services/groups-service/groups-service";
import { updateFavorites } from "../favorites/favorites-actions";
import { projectPanelActions, PROJECT_PANEL_CURRENT_UUID } from './project-panel-action';
import { Dispatch, MiddlewareAPI } from "redux";
import { ProjectResource } from "~/models/project";
import { updateResources } from "~/store/resources/resources-actions";
import { getProperty } from "~/store/properties/properties";
import { snackbarActions } from '../snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from '../data-explorer/data-explorer-reducer';
import { ListResults } from '~/services/common-service/common-resource-service';
import { loadContainers } from '../processes/processes-actions';
import { ResourceKind } from '~/models/resource';

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
                const response = await this.services.groupsService.contents(projectUuid, getParams(dataExplorer));
                api.dispatch<any>(updateFavorites(response.items.map(item => item.uuid)));
                api.dispatch(updateResources(response.items));
                await api.dispatch<any>(loadMissingProcessesInformation(response.items));
                api.dispatch(setItems(response));
            } catch (e) {
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

const setItems = (listResults: ListResults<GroupContentsResource>) =>
    projectPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer),
});

const getFilters = (dataExplorer: DataExplorer) => {
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

const getOrder = (dataExplorer: DataExplorer) => {
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
        message: 'Could not fetch project contents.'
    });

const projectPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Project panel is not ready.'
    });
