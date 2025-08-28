// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { Middleware } from 'redux';
import {
    dataExplorerActions,
    bindDataExplorerActions,
    DataTableRequestState,
    couldNotFetchItemsAvailable,
} from './data-explorer-action';
import { getDataExplorer } from './data-explorer-reducer';
import { DataExplorerMiddlewareService } from './data-explorer-middleware-service';

export const dataExplorerMiddleware =
    (service: DataExplorerMiddlewareService): Middleware =>
        (api) =>
            (next) => {
                const actions = bindDataExplorerActions(service.getId());

                return (action) => {
                    const handleAction =
                        <T extends { id: string }>(handler: (data: T) => void) =>
                            (data: T) => {
                                next(action);
                                if (data.id === service.getId()) {
                                    handler(data);
                                }
                            };
                    dataExplorerActions.match(action, {
                        SET_PAGE: handleAction(() => {
                            api.dispatch(actions.REQUEST_ITEMS(false));
                        }),
                        SET_ROWS_PER_PAGE: handleAction(() => {
                            api.dispatch(actions.REQUEST_ITEMS(true));
                        }),
                        SET_FILTERS: handleAction(() => {
                            api.dispatch(actions.RESET_PAGINATION());
                            api.dispatch(actions.SET_LOADING_ITEMS_AVAILABLE(true));
                            api.dispatch(actions.REQUEST_ITEMS(true));
                        }),
                        TOGGLE_SORT: handleAction(() => {
                            api.dispatch(actions.REQUEST_ITEMS(true));
                        }),
                        SET_EXPLORER_SEARCH_VALUE: handleAction(() => {
                            api.dispatch(actions.RESET_PAGINATION());
                            api.dispatch(actions.SET_LOADING_ITEMS_AVAILABLE(true));
                            api.dispatch(actions.REQUEST_ITEMS(true));
                        }),
                        REQUEST_ITEMS: handleAction(({ criteriaChanged = true, background }) => {
                            api.dispatch<any>(async (
                                dispatch: Dispatch,
                                getState: () => RootState,
                                services: ServiceRepository
                            ) => {
                                if (!background) { api.dispatch(actions.SET_WORKING(true)); }
                                while (true) {
                                    let de = getDataExplorer(
                                        getState().dataExplorer,
                                        service.getId()
                                    );
                                    switch (de.requestState) {
                                        case DataTableRequestState.IDLE:
                                            // Start a new request.
                                            try {
                                                dispatch(
                                                    actions.SET_REQUEST_STATE({
                                                        requestState: DataTableRequestState.PENDING,
                                                    })
                                                );

                                                // Fetch results
                                                const result = service.requestItems(api, criteriaChanged, background);

                                                // If criteria changed, fire off a count request
                                                if (criteriaChanged) {
                                                    dispatch(actions.REQUEST_COUNT(criteriaChanged, background));
                                                }

                                                // Await results
                                                await result;

                                            } catch {
                                                dispatch(
                                                    actions.SET_REQUEST_STATE({
                                                        requestState: DataTableRequestState.NEED_REFRESH,
                                                    })
                                                );
                                            }
                                            // Now check if the state is still PENDING, if it moved to NEED_REFRESH
                                            // then we need to reissue requestItems
                                            de = getDataExplorer(
                                                getState().dataExplorer,
                                                service.getId()
                                            );
                                            const complete =
                                                de.requestState === DataTableRequestState.PENDING;
                                            dispatch(
                                                actions.SET_REQUEST_STATE({
                                                    requestState: DataTableRequestState.IDLE,
                                                })
                                            );
                                            if (complete) {
                                                api.dispatch(actions.SET_WORKING(false));
                                                return;
                                            }
                                            break;
                                        case DataTableRequestState.PENDING:
                                            // State is PENDING, move it to NEED_REFRESH so that when the current request finishes it starts a new one.
                                            if (!background) {
                                                // Background refreshes are exempt from this behavior
                                                // because the data will already be up to date when the current request finishes
                                                // and to prevent refreshes from prolonging loading indicators of a non-background refresh
                                                dispatch(
                                                    actions.SET_REQUEST_STATE({
                                                        requestState: DataTableRequestState.NEED_REFRESH,
                                                    })
                                                );
                                            }
                                            return;
                                        case DataTableRequestState.NEED_REFRESH:
                                            // Nothing to do right now.
                                            return;
                                    }
                                }
                            });
                        }),
                        REQUEST_COUNT: handleAction(({ criteriaChanged = true, background }) => {
                            api.dispatch<any>(async (
                                dispatch: Dispatch,
                                getState: () => RootState,
                                services: ServiceRepository
                            ) => {
                                while (true) {
                                    let de = getDataExplorer(
                                        getState().dataExplorer,
                                        service.getId()
                                    );
                                    switch (de.countRequestState) {
                                        case DataTableRequestState.IDLE:
                                            // Start new count request
                                            dispatch(
                                                actions.SET_COUNT_REQUEST_STATE({
                                                    countRequestState: DataTableRequestState.PENDING,
                                                })
                                            );

                                            // Enable loading indicator on non-background fetches
                                            if (!background) {
                                                api.dispatch<any>(
                                                    dataExplorerActions.SET_LOADING_ITEMS_AVAILABLE({
                                                        id: service.getId(),
                                                        loadingItemsAvailable: true
                                                    })
                                                );
                                            }

                                            // Fetch count
                                            await service.requestCount(api, criteriaChanged, background)
                                                .catch(() => {
                                                    // Show error toast if count fetch failed
                                                    couldNotFetchItemsAvailable();
                                                })
                                                .finally(() => {
                                                    // Turn off itemsAvailable loading indicator when done
                                                    api.dispatch<any>(
                                                        dataExplorerActions.SET_LOADING_ITEMS_AVAILABLE({
                                                            id: service.getId(),
                                                            loadingItemsAvailable: false
                                                        })
                                                    );
                                                });

                                            // Now check if the state is still PENDING, if it moved to NEED_REFRESH
                                            // then we need to reissue requestCount
                                            de = getDataExplorer(
                                                getState().dataExplorer,
                                                service.getId()
                                            );
                                            const complete =
                                                de.countRequestState === DataTableRequestState.PENDING;
                                            dispatch(
                                                actions.SET_COUNT_REQUEST_STATE({
                                                    countRequestState: DataTableRequestState.IDLE,
                                                })
                                            );
                                            if (complete) {
                                                return;
                                            }
                                            break;
                                        case DataTableRequestState.PENDING:
                                            // State is PENDING, move it to NEED_REFRESH so that when the current request finishes it starts a new one.
                                            dispatch(
                                                actions.SET_COUNT_REQUEST_STATE({
                                                    countRequestState: DataTableRequestState.NEED_REFRESH,
                                                })
                                            );
                                            return;
                                        case DataTableRequestState.NEED_REFRESH:
                                            // Nothing to do right now.
                                            return;
                                    }
                                }
                            });
                        }),
                        default: () => next(action),
                    });
                };
            };
