// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, getOrder } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { resourcesActions } from 'store/resources/resources-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { collectionsContentAddressActions } from './collections-content-address-panel-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { updateFavorites } from 'store/favorites/favorites-actions';
import { updatePublicFavorites } from 'store/public-favorites/public-favorites-actions';
import { setBreadcrumbs } from '../breadcrumbs/breadcrumbs-actions';
import { ResourceKind, extractUuidKind } from 'models/resource';
import { ownerNameActions } from 'store/owner-name/owner-name-actions';
import { getUserDisplayName } from 'models/user';
import { CollectionResource } from 'models/collection';

export class CollectionsWithSameContentAddressMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(collectionPanelDataExplorerIsNotSet());
        } else {
            try {
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const userUuid = getUserUuid(api.getState());
                const pathname = api.getState().router.location!.pathname;
                const contentAddress = pathname.split('/')[2];
                const response = await this.services.collectionService.list({
                    limit: dataExplorer.rowsPerPage,
                    offset: dataExplorer.page * dataExplorer.rowsPerPage,
                    filters: new FilterBuilder()
                        .addEqual('portable_data_hash', contentAddress)
                        .addILike("name", dataExplorer.searchValue)
                        .getFilters(),
                    includeOldVersions: true,
                    order: getOrder<CollectionResource>(dataExplorer)
                });
                const userUuids = response.items.map(it => {
                    if (extractUuidKind(it.ownerUuid) === ResourceKind.USER) {
                        return it.ownerUuid;
                    } else {
                        return '';
                    }
                }
                );
                const groupUuids = response.items.map(it => {
                    if (extractUuidKind(it.ownerUuid) === ResourceKind.GROUP) {
                        return it.ownerUuid;
                    } else {
                        return '';
                    }
                });
                const responseUsers = await this.services.userService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', userUuids)
                        .getFilters(),
                    count: "none"
                });
                const responseGroups = await this.services.groupsService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', groupUuids)
                        .getFilters(),
                    count: "none"
                });
                responseUsers.items.forEach(it => {
                    api.dispatch<any>(ownerNameActions.SET_OWNER_NAME({
                        name: it.uuid === userUuid
                            ? 'User: Me'
                            : `User: ${getUserDisplayName(it)}`,
                        uuid: it.uuid
                    }));
                });
                responseGroups.items.forEach(it => {
                    api.dispatch<any>(ownerNameActions.SET_OWNER_NAME({ name: `Project: ${it.name}`, uuid: it.uuid }));
                });
                api.dispatch<any>(setBreadcrumbs([{ label: 'Projects', uuid: userUuid }]));
                api.dispatch<any>(updateFavorites(response.items.map(item => item.uuid)));
                api.dispatch<any>(updatePublicFavorites(response.items.map(item => item.uuid)));
                if (response.itemsAvailable === 1) {
                    api.dispatch<any>(navigateTo(response.items[0].uuid));
                    api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                } else {
                    api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                    api.dispatch(resourcesActions.SET_RESOURCES(response.items));
                    api.dispatch(collectionsContentAddressActions.SET_ITEMS({
                        items: response.items.map((resource: any) => resource.uuid),
                        itemsAvailable: response.itemsAvailable,
                        page: Math.floor(response.offset / response.limit),
                        rowsPerPage: response.limit
                    }));
                }
            } catch (e) {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(collectionsContentAddressActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchCollections());
            }
        }
    }
}

const collectionPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Collection panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchCollections = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch collection with this content address.',
        kind: SnackbarKind.ERROR
    });
