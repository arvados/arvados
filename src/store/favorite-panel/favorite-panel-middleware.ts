// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import { dataExplorerActions } from "../data-explorer/data-explorer-action";
import { favoriteService } from "../../services/services";
import { RootState } from "../store";
import { getDataExplorer } from "../data-explorer/data-explorer-reducer";
import { FilterBuilder } from "../../common/api/filter-builder";
import { DataColumns } from "../../components/data-table/data-table";
import {
    columns,
    FAVORITE_PANEL_ID,
    FavoritePanelColumnNames,
    FavoritePanelFilter
} from "../../views/favorite-panel/favorite-panel";
import { FavoritePanelItem, resourceToDataItem } from "../../views/favorite-panel/favorite-panel-item";
import { LinkResource } from "../../models/link";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";

export const favoritePanelMiddleware: Middleware = store => next => {
    next(dataExplorerActions.SET_COLUMNS({ id: FAVORITE_PANEL_ID, columns }));

    return action => {

        const handlePanelAction = <T extends { id: string }>(handler: (data: T) => void) =>
            (data: T) => {
                next(action);
                if (data.id === FAVORITE_PANEL_ID) {
                    handler(data);
                }
            };

        dataExplorerActions.match(action, {
            SET_PAGE: handlePanelAction(() => {
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: FAVORITE_PANEL_ID }));
            }),
            SET_ROWS_PER_PAGE: handlePanelAction(() => {
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: FAVORITE_PANEL_ID }));
            }),
            SET_FILTERS: handlePanelAction(() => {
                store.dispatch(dataExplorerActions.RESET_PAGINATION({ id: FAVORITE_PANEL_ID }));
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: FAVORITE_PANEL_ID }));
            }),
            TOGGLE_SORT: handlePanelAction(() => {
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: FAVORITE_PANEL_ID }));
            }),
            SET_SEARCH_VALUE: handlePanelAction(() => {
                store.dispatch(dataExplorerActions.RESET_PAGINATION({ id: FAVORITE_PANEL_ID }));
                store.dispatch(dataExplorerActions.REQUEST_ITEMS({ id: FAVORITE_PANEL_ID }));
            }),
            REQUEST_ITEMS: handlePanelAction(() => {
                const state = store.getState() as RootState;
                const dataExplorer = getDataExplorer(state.dataExplorer, FAVORITE_PANEL_ID);
                const columns = dataExplorer.columns as DataColumns<FavoritePanelItem, FavoritePanelFilter>;
                const typeFilters = getColumnFilters(columns, FavoritePanelColumnNames.TYPE);
                if (typeFilters.length > 0) {
                    favoriteService
                        .list(state.projects.currentItemId, {
                            limit: dataExplorer.rowsPerPage,
                            offset: dataExplorer.page * dataExplorer.rowsPerPage,
                            filters: FilterBuilder
                                .create<LinkResource>()
                                .addIsA("headUuid", typeFilters.map(filter => filter.type))
                                .addILike("name", dataExplorer.searchValue)
                        })
                        .then(response => {
                            store.dispatch(dataExplorerActions.SET_ITEMS({
                                id: FAVORITE_PANEL_ID,
                                items: response.items.map(resourceToDataItem),
                                itemsAvailable: response.itemsAvailable,
                                page: Math.floor(response.offset / response.limit),
                                rowsPerPage: response.limit
                            }));
                            store.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
                        });
                } else {
                    store.dispatch(dataExplorerActions.SET_ITEMS({
                        id: FAVORITE_PANEL_ID,
                        items: [],
                        itemsAvailable: 0,
                        page: 0,
                        rowsPerPage: dataExplorer.rowsPerPage
                    }));
                }
            }),
            default: () => next(action)
        });
    };
};

const getColumnFilters = (columns: DataColumns<FavoritePanelItem, FavoritePanelFilter>, columnName: string) => {
    const column = columns.find(c => c.name === columnName);
    return column && column.filters ? column.filters.filter(f => f.selected) : [];
};



