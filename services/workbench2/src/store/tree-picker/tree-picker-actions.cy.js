// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { createServices } from "services/services";
import { configureStore } from "../store";
import { createBrowserHistory } from "history";
import { mockConfig } from 'common/config';
import Axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { ResourceKind } from 'models/resource';
import { SHARED_PROJECT_ID, initProjectsTreePicker } from "./tree-picker-actions";
import { CollectionFileType } from "models/collection-file";

describe('tree-picker-actions', () => {
    const axiosInst = Axios.create({ headers: {} });
    const axiosMock = new MockAdapter(axiosInst);

    let store;
    let services;
    const config = {};
    const actions = {
        progressFn: (id, working) => { },
        errorFn: (id, message) => { }
    };
    let importMocks;

    beforeEach(() => {
        axiosMock.reset();
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
        localStorage.clear();
        importMocks = [];
    });

    afterEach(() => {
        importMocks.map(m => m.restore());
    });

    it('initializes preselected tree picker nodes', async () => {
        const dispatchMock = cy.stub();
        const dispatchWrapper = (action) => {
            dispatchMock(action);
            return store.dispatch(action);
        };

        const emptyCollectionUuid = "zzzzz-4zz18-000000000000000";
        const collectionUuid = "zzzzz-4zz18-111111111111111";
        const parentProjectUuid = "zzzzz-j7d0g-000000000000000";
        const childCollectionUuid = "zzzzz-4zz18-222222222222222";

        const fakeResources = {
            [emptyCollectionUuid]: {
                kind: ResourceKind.COLLECTION,
                ownerUuid: '',
                files: [],
            },
            [collectionUuid]: {
                kind: ResourceKind.COLLECTION,
                ownerUuid: '',
                files: [{
                    id: `${collectionUuid}/directory`,
                    name: "directory",
                    path: "",
                    type: CollectionFileType.DIRECTORY,
                    url: `/c=${collectionUuid}/directory/`,
                }]
            },
            [parentProjectUuid]: {
                kind: ResourceKind.GROUP,
                ownerUuid: '',
            },
            [childCollectionUuid]: {
                kind: ResourceKind.COLLECTION,
                ownerUuid: parentProjectUuid,
                files: [
                    {
                        id: `${childCollectionUuid}/mainDir`,
                        name: "mainDir",
                        path: "",
                        type: CollectionFileType.DIRECTORY,
                        url: `/c=${childCollectionUuid}/mainDir/`,
                    },
                    {
                        id: `${childCollectionUuid}/mainDir/subDir`,
                        name: "subDir",
                        path: "/mainDir",
                        type: CollectionFileType.DIRECTORY,
                        url: `/c=${childCollectionUuid}/mainDir/subDir`,
                    }
                ],
            },
        };

        services.ancestorsService.ancestors = cy.stub().callsFake((startUuid, endUuid) => {
            let ancestors = [];
            let uuid = startUuid;
            while (uuid?.length && fakeResources[uuid]) {
                const resource = fakeResources[uuid];
                if (resource.kind === ResourceKind.COLLECTION) {
                    ancestors.unshift({
                        uuid, kind: resource.kind,
                        ownerUuid: resource.ownerUuid,
                    });
                } else if (resource.kind === ResourceKind.GROUP) {
                    ancestors.unshift({
                        uuid, kind: resource.kind,
                        ownerUuid: resource.ownerUuid,
                    });
                }
                uuid = resource.ownerUuid;
            }
            return ancestors;
        });

        services.collectionService.files = cy.stub(async (uuid)=> {
            return fakeResources[uuid]?.files || [];
        });

        services.groupsService.contents = cy.stub(async (uuid, args) => {
            const items = Object.keys(fakeResources).map(uuid => ({...fakeResources[uuid], uuid})).filter(item => item.ownerUuid === uuid);
            return {items: items, itemsAvailable: items.length};
        });

        const pickerId = "pickerId";

        // When collection preselected
        await initProjectsTreePicker(pickerId, {
            selectedItemUuids: [emptyCollectionUuid],
            includeDirectories: true,
            includeFiles: false,
            multi: true,
        })(dispatchWrapper, store.getState, services);

        // Expect ancestor service to be called
        expect(services.ancestorsService.ancestors).to.be.calledWith(emptyCollectionUuid, '');
        // Expect top level to be expanded and node to be selected
        console.log(store.getState().treePicker["pickerId_shared"]);
        expect(store.getState().treePicker["pickerId_shared"][SHARED_PROJECT_ID].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][emptyCollectionUuid].selected).to.equal(true);


        // When collection subdirectory is preselected
        await initProjectsTreePicker(pickerId, {
            selectedItemUuids: [`${collectionUuid}/directory`],
            includeDirectories: true,
            includeFiles: false,
            multi: true,
        })(dispatchWrapper, store.getState, services);

        // Expect ancestor service to be called
        expect(services.ancestorsService.ancestors).to.be.calledWith(collectionUuid, '');
        // Expect top level to be expanded and node to be selected
        expect(store.getState().treePicker["pickerId_shared"][SHARED_PROJECT_ID].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][collectionUuid].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][collectionUuid].selected).to.equal(false);
        expect(store.getState().treePicker["pickerId_shared"][`${collectionUuid}/directory`].selected).to.equal(true);


        // When subdirectory of collection inside project is preselected
        await initProjectsTreePicker(pickerId, {
            selectedItemUuids: [`${childCollectionUuid}/mainDir/subDir`],
            includeDirectories: true,
            includeFiles: false,
            multi: true,
        })(dispatchWrapper, store.getState, services);

        // Expect ancestor service to be called
        expect(services.ancestorsService.ancestors).to.be.calledWith(childCollectionUuid, '');
        // Expect parent project and collection to be expanded
        expect(store.getState().treePicker["pickerId_shared"][SHARED_PROJECT_ID].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][parentProjectUuid].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][parentProjectUuid].selected).to.equal(false);
        expect(store.getState().treePicker["pickerId_shared"][childCollectionUuid].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][childCollectionUuid].selected).to.equal(false);
        // Expect main directory to be expanded
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir`].expanded).to.equal(true);
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir`].selected).to.equal(false);
        // Expect sub directory to be selected
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir/subDir`].expanded).to.equal(false);
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir/subDir`].selected).to.equal(true);


    });
});
