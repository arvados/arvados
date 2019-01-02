
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import { dataExplorerActions, bindDataExplorerActions } from "./data-explorer-action";
import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";

export const dataExplorerMiddleware = (service: DataExplorerMiddlewareService): Middleware => api => next => {
    const actions = bindDataExplorerActions(service.getId());

    return action => {
        const handleAction = <T extends { id: string }>(handler: (data: T) => void) =>
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
                service.requestItems(api, criteriaChanged);
            }),
            default: () => next(action)
        });
    };
};
