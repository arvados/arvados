// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import Axios from "axios";
import { mockConfig } from "common/config";
import { createServices } from "services/services";
import { createBrowserHistory } from "history";
import { Provider } from "react-redux";
import { configureStore } from "store/store";
import { TreePicker } from "./tree-picker";
import { initUserProject, receiveTreePickerData, extractGroupContentsNodeData } from "store/tree-picker/tree-picker-actions";
import { authActions } from "store/auth/auth-action";
import { ResourceKind } from "models/resource";
import { updateResources } from "store/resources/resources-actions";
import { CustomTheme } from "common/custom-theme";
import { ThemeProvider } from "@mui/material";

describe('<TreePicker />', () => {
    let store;
    let services;
    const axiosInst = Axios.create({ headers: {} });
    const config = {};
    const actions = {
        progressFn: (id, working) => { },
        errorFn: (id, message) => { }
    };
    const TEST_PICKER_ID = 'testPickerId';
    const fakeUser = {
        email: "test@test.com",
        firstName: "John",
        lastName: "Doe",
        uuid: "zzzzz-tpzed-xurymjxw79nv3jz",
        ownerUuid: "ownerUuid",
        username: "username",
        prefs: {},
        isAdmin: false,
        isActive: true,
        canWrite: false,
        canManage: false,
    };
    const renderItem = (item) => (
        <li data-id={item.id}>{item.data.name}</li>
    );

    beforeEach(() => {
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
        store.dispatch(authActions.USER_DETAILS_SUCCESS(fakeUser));
        store.dispatch(initUserProject(TEST_PICKER_ID));
    });

    it("renders tree picker with initial home project state", () => {
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <TreePicker
                    pickerId={TEST_PICKER_ID}
                    render={renderItem}
                    onContextMenu={() => {}}
                    toggleItemOpen={() => {}}
                    toggleItemActive={() => {}}
                    toggleItemSelection={() => {}}
                />
                </ThemeProvider>
            </Provider>);

        cy.get(`li[data-id="${fakeUser.uuid}"]`).should('have.text', 'Home Projects');
    });

    it("displays item loaded into treePicker store", () => {
        const fakeProject = {
            uuid: "zzzzz-j7d0g-111111111111111",
            name: "FakeProject",
            kind: ResourceKind.PROJECT,
        };

        store.dispatch(receiveTreePickerData({
            id: fakeUser.uuid,
            pickerId: TEST_PICKER_ID,
            data: [fakeProject],
            extractNodeData: extractGroupContentsNodeData(false)
        }));

        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <TreePicker
                    pickerId={TEST_PICKER_ID}
                    render={renderItem}
                    onContextMenu={() => {}}
                    toggleItemOpen={() => {}}
                    toggleItemActive={() => {}}
                    toggleItemSelection={() => {}}
                />
                </ThemeProvider>
            </Provider>);

        cy.get(`li[data-id="${fakeUser.uuid}"]`).should('have.text', 'Home Projects');
        cy.get(`[data-id="${fakeProject.uuid}"]`).should('have.text', 'FakeProject');
    });

    it("preserves treenode name when exists in resources", () => {
        const treeProjectResource = {
            uuid: "zzzzz-j7d0g-111111111111111",
            name: "FakeProject",
            kind: ResourceKind.PROJECT,
        };
        const treeProjectResource2 = {
            uuid: "zzzzz-j7d0g-222222222222222",
            name: "",
            kind: ResourceKind.PROJECT,
        };

        const storeProjectResource = {
            ...treeProjectResource,
            name: "StoreProjectName",
            description: "Test description",
        };
        const storeProjectResource2 = {
            ...treeProjectResource2,
            name: "StoreProjectName2",
            description: "Test description",
        };

        store.dispatch(updateResources([storeProjectResource, storeProjectResource2]));
        store.dispatch(receiveTreePickerData({
            id: fakeUser.uuid,
            pickerId: TEST_PICKER_ID,
            data: [treeProjectResource, treeProjectResource2],
            extractNodeData: extractGroupContentsNodeData(false)
        }));

        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <TreePicker
                    pickerId={TEST_PICKER_ID}
                    render={renderItem}
                    onContextMenu={() => {}}
                    toggleItemOpen={() => {}}
                    toggleItemActive={() => {}}
                    toggleItemSelection={() => {}}
                />
                </ThemeProvider>
            </Provider>);

        cy.get(`[data-id="${fakeUser.uuid}"]`).should('have.text', 'Home Projects');
        cy.get(`[data-id="${treeProjectResource.uuid}"]`).should('have.text', 'FakeProject');
        cy.get(`[data-id="${treeProjectResource2.uuid}"]`).should('have.text', '');
    });

});
