
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import { dataExplorerActions, bindDataExplorerActions } from "./data-explorer-action";
import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";

export const dataExplorerMiddleware = (service: DataExplorerMiddlewareService): Middleware => api => next => {
    const actions = bindDataExplorerActions(service.getId());
    next(actions.SET_COLUMNS({ columns: service.getColumns() }));

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
                api.dispatch(actions.REQUEST_ITEMS());
            }),
            SET_ROWS_PER_PAGE: handleAction(() => {
                api.dispatch(actions.REQUEST_ITEMS());
            }),
            SET_FILTERS: handleAction(() => {
                api.dispatch(actions.RESET_PAGINATION());
                api.dispatch(actions.REQUEST_ITEMS());
            }),
            TOGGLE_SORT: handleAction(() => {
                api.dispatch(actions.REQUEST_ITEMS());
            }),
            SET_SEARCH_VALUE: handleAction(() => {
                api.dispatch(actions.RESET_PAGINATION());
                api.dispatch(actions.REQUEST_ITEMS());
            }),
            REQUEST_ITEMS: handleAction(() => {
                service.requestItems(api);
            }),
            default: () => next(action)
        });
    };
};
