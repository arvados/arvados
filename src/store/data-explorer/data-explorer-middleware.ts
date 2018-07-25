
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "../../../node_modules/redux";
import { dataExplorerActions } from "./data-explorer-action";
import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";

export const dataExplorerMiddleware = (service: DataExplorerMiddlewareService): Middleware => api => next => {
    service.Api = api;
    next(dataExplorerActions.SET_COLUMNS({ id: service.Id, columns: service.Columns }));
    return action => {
        const handleAction = <T extends { id: string }>(handler: (data: T) => void) =>
            (data: T) => {
                next(action);
                if (data.id === service.Id) {
                    handler(data);
                }
            };
        dataExplorerActions.match(action, {
            SET_PAGE: handleAction(() => {
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: service.Id }));
            }),
            SET_ROWS_PER_PAGE: handleAction(() => {
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: service.Id }));
            }),
            SET_FILTERS: handleAction(() => {
                api.dispatch(dataExplorerActions.RESET_PAGINATION({ id: service.Id }));
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: service.Id }));
            }),
            TOGGLE_SORT: handleAction(() => {
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: service.Id }));
            }),
            SET_SEARCH_VALUE: handleAction(() => {
                api.dispatch(dataExplorerActions.RESET_PAGINATION({ id: service.Id }));
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: service.Id }));
            }),
            REQUEST_ITEMS: handleAction(() => {
                service.requestItems(api);
            }),
            default: () => next(action)
        });
    };
};
