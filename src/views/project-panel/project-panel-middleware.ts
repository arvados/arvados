// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import actions from "../../store/data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID, columns } from "./project-panel";
import { groupsService } from "../../services/services";
import { RootState } from "../../store/store";
import { getDataExplorer } from "../../store/data-explorer/data-explorer-reducer";
import { resourceToDataItem } from "./project-panel-item";

export const projectPanelMiddleware: Middleware = store => next => {
    next(actions.SET_COLUMNS({ id: PROJECT_PANEL_ID, columns }));

    return action => {

        const handleProjectPanelAction = <T extends { id: string }>(handler: (data: T) => void) =>
            (data: T) => {
                next(action);
                if (data.id === PROJECT_PANEL_ID) {
                    handler(data);
                }
            };

        actions.match(action, {
            SET_PAGE: handleProjectPanelAction(() => {
                store.dispatch(actions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            SET_ROWS_PER_PAGE: handleProjectPanelAction(() => {
                store.dispatch(actions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            }),
            REQUEST_ITEMS: handleProjectPanelAction(() => {
                const state = store.getState() as RootState;
                const dataExplorer = getDataExplorer(state.dataExplorer, PROJECT_PANEL_ID);
                groupsService
                    .contents(state.projects.currentItemId, {
                        limit: dataExplorer.rowsPerPage,
                        offset: dataExplorer.page * dataExplorer.rowsPerPage,
                    })
                    .then(response => {
                        store.dispatch(actions.SET_ITEMS({
                            id: PROJECT_PANEL_ID,
                            items: response.items.map(resourceToDataItem),
                            itemsAvailable: response.itemsAvailable,
                            page: Math.floor(response.offset / response.limit),
                            rowsPerPage: response.limit
                        }));
                    });

            }),
            default: () => next(action)
        });
    };
};
