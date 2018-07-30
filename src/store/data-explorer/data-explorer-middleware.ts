
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "../../../node_modules/redux";
import { dataExplorerActions } from "./data-explorer-action";
import { DataExplorerMiddlewareService } from "./data-explorer-middleware-service";

export const dataExplorerMiddleware = (dataExplorerService: DataExplorerMiddlewareService): Middleware => api => next => {
    dataExplorerService.setApi(api);
    next(dataExplorerActions.SET_COLUMNS({ id: dataExplorerService.getId(), columns: dataExplorerService.getColumns() }));
    return action => {
        const handleAction = <T extends { id: string }>(handler: (data: T) => void) =>
            (data: T) => {
                next(action);
                if (data.id === dataExplorerService.getId()) {
                    handler(data);
                }
            };
        dataExplorerActions.match(action, {
            SET_PAGE: handleAction(() => {
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: dataExplorerService.getId() }));
            }),
            SET_ROWS_PER_PAGE: handleAction(() => {
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: dataExplorerService.getId() }));
            }),
            SET_FILTERS: handleAction(() => {
                api.dispatch(dataExplorerActions.RESET_PAGINATION({ id: dataExplorerService.getId() }));
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: dataExplorerService.getId() }));
            }),
            TOGGLE_SORT: handleAction(() => {
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: dataExplorerService.getId() }));
            }),
            SET_SEARCH_VALUE: handleAction(() => {
                api.dispatch(dataExplorerActions.RESET_PAGINATION({ id: dataExplorerService.getId() }));
                api.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: dataExplorerService.getId() }));
            }),
            REQUEST_ITEMS: handleAction(() => {
                dataExplorerService.requestItems(api);
            }),
            default: () => next(action)
        });
    };
};
