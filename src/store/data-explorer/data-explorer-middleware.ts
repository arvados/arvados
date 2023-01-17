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
                    api.dispatch(actions.REQUEST_ITEMS(true));
                }),
                TOGGLE_SORT: handleAction(() => {
                    api.dispatch(actions.REQUEST_ITEMS(true));
                }),
                SET_EXPLORER_SEARCH_VALUE: handleAction(() => {
                    api.dispatch(actions.RESET_PAGINATION());
                    api.dispatch(actions.REQUEST_ITEMS(true));
                }),
                REQUEST_ITEMS: handleAction(({ criteriaChanged }) => {
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
                            switch (de.requestState) {
                                case DataTableRequestState.IDLE:
                                    // Start a new request.
                                    try {
                                        dispatch(
                                            actions.SET_REQUEST_STATE({
                                                requestState: DataTableRequestState.PENDING,
                                            })
                                        );
                                        await service.requestItems(api, criteriaChanged);
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
                                        return;
                                    }
                                    break;
                                case DataTableRequestState.PENDING:
                                    // State is PENDING, move it to NEED_REFRESH so that when the current request finishes it starts a new one.
                                    dispatch(
                                        actions.SET_REQUEST_STATE({
                                            requestState: DataTableRequestState.NEED_REFRESH,
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
