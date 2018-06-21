// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import projectsReducer from "./project-reducer";
import actions from "./project-action";

describe('project-reducer', () => {
    it('should add new project to the list', () => {
        const initialState = undefined;
        const project = {
            name: 'test',
            href: 'href',
            createdAt: '2018-01-01',
            modifiedAt: '2018-01-01',
            ownerUuid: 'owner-test123',
            uuid: 'test123'
        };

        const state = projectsReducer(initialState, actions.CREATE_PROJECT(project));
        expect(state).toEqual([project]);
    });

    it('should load projects', () => {
        const initialState = undefined;
        const project = {
            name: 'test',
            href: 'href',
            createdAt: '2018-01-01',
            modifiedAt: '2018-01-01',
            ownerUuid: 'owner-test123',
            uuid: 'test123'
        };

        const projects = [project, project];
        const state = projectsReducer(initialState, actions.PROJECTS_SUCCESS({ projects, parentItemId: undefined }));
        expect(state).toEqual([{
            active: false,
            open: false,
            id: "test123",
            items: [],
            data: project,
            status: 0
        }, {
            active: false,
            open: false,
            id: "test123",
            items: [],
            data: project,
            status: 0
        }
        ]);
    });

    it('should remove activity on projects list', () => {
        const initialState = [
            {
                data: {
                    name: 'test',
                    href: 'href',
                    createdAt: '2018-01-01',
                    modifiedAt: '2018-01-01',
                    ownerUuid: 'owner-test123',
                    uuid: 'test123',
                },
                id: "1",
                open: true,
                active: true,
                status: 1
            }
        ];
        const project = [
            {
                data: {
                    name: 'test',
                    href: 'href',
                    createdAt: '2018-01-01',
                    modifiedAt: '2018-01-01',
                    ownerUuid: 'owner-test123',
                    uuid: 'test123',
                },
                id: "1",
                open: true,
                active: false,
                status: 1
            }
        ];

        const state = projectsReducer(initialState, actions.RESET_PROJECT_TREE_ACTIVITY(initialState[0].id));
        expect(state).toEqual(project);
    });

    it('should toggle project tree item activity', () => {
        const initialState = [
            {
                data: {
                    name: 'test',
                    href: 'href',
                    createdAt: '2018-01-01',
                    modifiedAt: '2018-01-01',
                    ownerUuid: 'owner-test123',
                    uuid: 'test123',
                },
                id: "1",
                open: true,
                active: false,
                status: 1
            }
        ];
        const project = [
            {
                data: {
                    name: 'test',
                    href: 'href',
                    createdAt: '2018-01-01',
                    modifiedAt: '2018-01-01',
                    ownerUuid: 'owner-test123',
                    uuid: 'test123',
                },
                id: "1",
                open: true,
                active: true,
                status: 1
            }
        ];

        const state = projectsReducer(initialState, actions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(initialState[0].id));
        expect(state).toEqual(project);
    });


    it('should close project tree item ', () => {
        const initialState = [
            {
                data: {
                    name: 'test',
                    href: 'href',
                    createdAt: '2018-01-01',
                    modifiedAt: '2018-01-01',
                    ownerUuid: 'owner-test123',
                    uuid: 'test123',
                },
                id: "1",
                open: true,
                active: false,
                status: 1,
                toggled: false,
            }
        ];
        const project = [
            {
                data: {
                    name: 'test',
                    href: 'href',
                    createdAt: '2018-01-01',
                    modifiedAt: '2018-01-01',
                    ownerUuid: 'owner-test123',
                    uuid: 'test123',
                },
                id: "1",
                open: false,
                active: false,
                status: 1,
                toggled: true
            }
        ];

        const state = projectsReducer(initialState, actions.TOGGLE_PROJECT_TREE_ITEM_OPEN(initialState[0].id));
        expect(state).toEqual(project);
    });
});
