// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import projectsReducer, { getTreePath } from "./project-reducer";
import actions from "./project-action";
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { ResourceKind } from "../../models/resource";

describe('project-reducer', () => {
    it('should add new project to the list', () => {
        const initialState = undefined;
        const project = {
            name: 'test',
            href: 'href',
            createdAt: '2018-01-01',
            modifiedAt: '2018-01-01',
            ownerUuid: 'owner-test123',
            uuid: 'test123',
            kind: ResourceKind.PROJECT
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
            uuid: 'test123',
            kind: ResourceKind.PROJECT
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
});

describe("findTreeBranch", () => {

    const createTreeItem = (id: string, items?: Array<TreeItem<string>>): TreeItem<string> => ({
        id,
        items,
        active: false,
        data: "",
        open: false,
        status: TreeItemStatus.Initial
    });

    it("should return an array that matches path to the given item", () => {
        const tree: Array<TreeItem<string>> = [
            createTreeItem("1", [
                createTreeItem("1.1", [
                    createTreeItem("1.1.1"),
                    createTreeItem("1.1.2")
                ])
            ]),
            createTreeItem("2", [
                createTreeItem("2.1", [
                    createTreeItem("2.1.1"),
                    createTreeItem("2.1.2")
                ])
            ])
        ];
        const branch = getTreePath(tree, "2.1.1");
        expect(branch.map(item => item.id)).toEqual(["2", "2.1", "2.1.1"]);
    });

    it("should return empty array if item is not found", () => {
        const tree: Array<TreeItem<string>> = [
            createTreeItem("1", [
                createTreeItem("1.1", [
                    createTreeItem("1.1.1"),
                    createTreeItem("1.1.2")
                ])
            ]),
            createTreeItem("2", [
                createTreeItem("2.1", [
                    createTreeItem("2.1.1"),
                    createTreeItem("2.1.2")
                ])
            ])
        ];
        expect(getTreePath(tree, "3")).toHaveLength(0);
    });

});
