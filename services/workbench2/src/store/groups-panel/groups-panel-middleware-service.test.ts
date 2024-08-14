// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios, { AxiosInstance, AxiosResponse } from "axios";
import { mockConfig } from "common/config";
import { createBrowserHistory } from "history";
import { GroupsPanelMiddlewareService } from "./groups-panel-middleware-service";
import { dataExplorerMiddleware } from "store/data-explorer/data-explorer-middleware";
import { Dispatch, MiddlewareAPI } from "redux";
import { DataColumns } from "components/data-table/data-table";
import { dataExplorerActions } from "store/data-explorer/data-explorer-action";
import { SortDirection } from "components/data-table/data-column";
import { createTree } from 'models/tree';
import { DataTableFilterItem } from "components/data-table-filters/data-table-filters-tree";
import { GROUPS_PANEL_ID } from "./groups-panel-actions";
import { RootState, RootStore, configureStore } from "store/store";
import { ServiceRepository, createServices } from "services/services";
import { ApiActions } from "services/api/api-actions";
import { ListResults } from "services/common-service/common-service";
import { GroupResource } from "models/group";
import { getResource } from "store/resources/resources";

describe("GroupsPanelMiddlewareService", () => {
    let axiosInst: AxiosInstance;
    let store: RootStore;
    let services: ServiceRepository;
    const config: any = {};
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };

    beforeEach(() => {
        axiosInst = Axios.create({ headers: {} });
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
    });

    it("requests group member counts and updates resource store", async () => {
        // Given
        const fakeUuid = "zzzzz-j7d0g-000000000000000";
        axiosInst.get = jest.fn((url: string) => {
            if (url === '/groups') {
                return Promise.resolve(
                    { data: {
                        kind: "",
                        offset: 0,
                        limit: 100,
                        items: [{
                            can_manage: true,
                            can_write: true,
                            created_at: "2023-11-15T20:57:01.723043000Z",
                            delete_at: null,
                            description: null,
                            etag: "0000000000000000000000000",
                            frozen_by_uuid: null,
                            group_class: "role",
                            href: `/groups/${fakeUuid}`,
                            is_trashed: false,
                            kind: "arvados#group",
                            modified_at: "2023-11-15T20:57:01.719986000Z",
                            modified_by_user_uuid: "zzzzz-tpzed-000000000000000",
                            name: "Test Group",
                            owner_uuid: "zzzzz-tpzed-000000000000000",
                            properties: {},
                            trash_at: null,
                            uuid: fakeUuid,
                            writable_by: [
                                "zzzzz-tpzed-000000000000000",
                            ]
                        }],
                        items_available: 1,
                    }} as AxiosResponse);
            } else if (url === '/links') {
                return Promise.resolve(
                    { data: {
                        items: [],
                        items_available: 234,
                        kind: "arvados#linkList",
                        limit: 0,
                        offset: 0
                    }} as AxiosResponse);
            } else {
                return Promise.resolve(
                    { data: {}} as AxiosResponse);
            }
        }) as AxiosInstance['get'];

        // When
        await store.dispatch(dataExplorerActions.REQUEST_ITEMS({id: GROUPS_PANEL_ID}));
        // Wait for async fetching of group count promises to resolve
        await new Promise(setImmediate);

        // Expect
        expect(axiosInst.get).toHaveBeenCalledTimes(2);
        expect(axiosInst.get).toHaveBeenCalledWith('/groups', expect.anything());
        expect(axiosInst.get).toHaveBeenCalledWith('/links', expect.anything());
        const group = getResource<GroupResource>(fakeUuid)(store.getState().resources);
        expect(group?.memberCount).toBe(234);
    });

    it('requests group member count and stores null on failure', async () => {
        // Given
        const fakeUuid = "zzzzz-j7d0g-000000000000000";
        axiosInst.get = jest.fn((url: string) => {
            if (url === '/groups') {
                return Promise.resolve(
                    { data: {
                        kind: "",
                        offset: 0,
                        limit: 100,
                        items: [{
                            can_manage: true,
                            can_write: true,
                            created_at: "2023-11-15T20:57:01.723043000Z",
                            delete_at: null,
                            description: null,
                            etag: "0000000000000000000000000",
                            frozen_by_uuid: null,
                            group_class: "role",
                            href: `/groups/${fakeUuid}`,
                            is_trashed: false,
                            kind: "arvados#group",
                            modified_at: "2023-11-15T20:57:01.719986000Z",
                            modified_by_user_uuid: "zzzzz-tpzed-000000000000000",
                            name: "Test Group",
                            owner_uuid: "zzzzz-tpzed-000000000000000",
                            properties: {},
                            trash_at: null,
                            uuid: fakeUuid,
                            writable_by: [
                                "zzzzz-tpzed-000000000000000",
                            ]
                        }],
                        items_available: 1,
                    }} as AxiosResponse);
            } else if (url === '/links') {
                return Promise.reject();
            } else {
                return Promise.resolve({ data: {}} as AxiosResponse);
            }
        }) as AxiosInstance['get'];

        // When
        await store.dispatch(dataExplorerActions.REQUEST_ITEMS({id: GROUPS_PANEL_ID}));
        // Wait for async fetching of group count promises to resolve
        await new Promise(setImmediate);

        // Expect
        expect(axiosInst.get).toHaveBeenCalledTimes(2);
        expect(axiosInst.get).toHaveBeenCalledWith('/groups', expect.anything());
        expect(axiosInst.get).toHaveBeenCalledWith('/links', expect.anything());
        const group = getResource<GroupResource>(fakeUuid)(store.getState().resources);
        expect(group?.memberCount).toBe(null);
    });

});
