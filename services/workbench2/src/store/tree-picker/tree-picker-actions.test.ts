// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository, createServices } from "services/services";
import { configureStore, RootStore } from "../store";
import { createBrowserHistory } from "history";
import { mockConfig } from 'common/config';
import { ApiActions } from "services/api/api-actions";
import Axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { ResourceKind } from 'models/resource';
import { SHARED_PROJECT_ID, initProjectsTreePicker } from "./tree-picker-actions";
import { CollectionResource } from "models/collection";
import { GroupResource } from "models/group";
import { CollectionDirectory, CollectionFile, CollectionFileType } from "models/collection-file";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { ListResults } from "services/common-service/common-service";

describe('tree-picker-actions', () => {
    const axiosInst = Axios.create({ headers: {} });
    const axiosMock = new MockAdapter(axiosInst);

    let store: RootStore;
    let services: ServiceRepository;
    const config: any = {};
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };
    let importMocks: any[];

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
        const dispatchMock = jest.fn();
        const dispatchWrapper = (action: any) => {
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

        services.ancestorsService.ancestors = jest.fn(async (startUuid, endUuid) => {
            let ancestors: (GroupResource | CollectionResource)[] = [];
            let uuid = startUuid;
            while (uuid?.length && fakeResources[uuid]) {
                const resource = fakeResources[uuid];
                if (resource.kind === ResourceKind.COLLECTION) {
                    ancestors.unshift({
                        uuid, kind: resource.kind,
                        ownerUuid: resource.ownerUuid,
                    } as CollectionResource);
                } else if (resource.kind === ResourceKind.GROUP) {
                    ancestors.unshift({
                        uuid, kind: resource.kind,
                        ownerUuid: resource.ownerUuid,
                    } as GroupResource);
                }
                uuid = resource.ownerUuid;
            }
            return ancestors;
        });

        services.collectionService.files = jest.fn(async (uuid): Promise<(CollectionDirectory | CollectionFile)[]> => {
            return fakeResources[uuid]?.files || [];
        });

        services.groupsService.contents = jest.fn(async (uuid, args) => {
            const items = Object.keys(fakeResources).map(uuid => ({...fakeResources[uuid], uuid})).filter(item => item.ownerUuid === uuid);
            return {items: items as GroupContentsResource[], itemsAvailable: items.length} as ListResults<GroupContentsResource>;
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
        expect(services.ancestorsService.ancestors).toHaveBeenCalledWith(emptyCollectionUuid, '');
        // Expect top level to be expanded and node to be selected
        expect(store.getState().treePicker["pickerId_shared"][SHARED_PROJECT_ID].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][emptyCollectionUuid].selected).toBe(true);


        // When collection subdirectory is preselected
        await initProjectsTreePicker(pickerId, {
            selectedItemUuids: [`${collectionUuid}/directory`],
            includeDirectories: true,
            includeFiles: false,
            multi: true,
        })(dispatchWrapper, store.getState, services);

        // Expect ancestor service to be called
        expect(services.ancestorsService.ancestors).toHaveBeenCalledWith(collectionUuid, '');
        // Expect top level to be expanded and node to be selected
        expect(store.getState().treePicker["pickerId_shared"][SHARED_PROJECT_ID].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][collectionUuid].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][collectionUuid].selected).toBe(false);
        expect(store.getState().treePicker["pickerId_shared"][`${collectionUuid}/directory`].selected).toBe(true);


        // When subdirectory of collection inside project is preselected
        await initProjectsTreePicker(pickerId, {
            selectedItemUuids: [`${childCollectionUuid}/mainDir/subDir`],
            includeDirectories: true,
            includeFiles: false,
            multi: true,
        })(dispatchWrapper, store.getState, services);

        // Expect ancestor service to be called
        expect(services.ancestorsService.ancestors).toHaveBeenCalledWith(childCollectionUuid, '');
        // Expect parent project and collection to be expanded
        expect(store.getState().treePicker["pickerId_shared"][SHARED_PROJECT_ID].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][parentProjectUuid].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][parentProjectUuid].selected).toBe(false);
        expect(store.getState().treePicker["pickerId_shared"][childCollectionUuid].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][childCollectionUuid].selected).toBe(false);
        // Expect main directory to be expanded
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir`].expanded).toBe(true);
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir`].selected).toBe(false);
        // Expect sub directory to be selected
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir/subDir`].expanded).toBe(false);
        expect(store.getState().treePicker["pickerId_shared"][`${childCollectionUuid}/mainDir/subDir`].selected).toBe(true);


    });
});
