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
        const state = projectsReducer(initialState, actions.PROJECTS_SUCCESS({projects, parentItemId: undefined}));
        expect(state).toEqual([{
                active: false,
                open: false,
                id: "test123",
                items: [],
                data: project
            }, {
                active: false,
                open: false,
                id: "test123",
                items: [],
                data: project
            }
        ]);
    });
});
