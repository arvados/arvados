// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, getNodeChildrenIds, getNode, TreeNodeStatus } from 'models/tree';
import { pipe } from 'lodash/fp';
import { treePickerReducer } from "./tree-picker-reducer";
import { treePickerActions } from "./tree-picker-actions";
import { initTreeNode } from 'models/tree';

describe('TreePickerReducer', () => {
    it('LOAD_TREE_PICKER_NODE - initial state', () => {
        const tree = createTree();
        const newState = treePickerReducer({}, treePickerActions.LOAD_TREE_PICKER_NODE({ id: '1', pickerId: "projects" }));
        expect(newState).to.deep.equal({ 'projects': tree });
    });

    it('LOAD_TREE_PICKER_NODE', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE({ id: '1', pickerId: "projects" }))
        )({ projects: createTree() });

        expect(getNode('1')(newState.projects)).to.deep.equal({
            ...initTreeNode({ id: '1', value: '1' }),
            status: TreeNodeStatus.PENDING
        });
    });

    it('LOAD_TREE_PICKER_NODE_SUCCESS - initial state', () => {
        const subNode = initTreeNode({ id: '1.1', value: '1.1' });
        const newState = treePickerReducer({}, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [subNode], pickerId: "projects" }));
        expect(getNodeChildrenIds('')(newState.projects)).to.deep.equal(['1.1']);
    });

    it('LOAD_TREE_PICKER_NODE_SUCCESS', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const subNode = initTreeNode({ id: '1.1', value: '1.1' });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '1', nodes: [subNode], pickerId: "projects" }))
        )({ projects: createTree() });
        expect(getNodeChildrenIds('1')(newState.projects)).to.deep.equal(['1.1']);
        expect(getNode('1')(newState.projects)).to.deep.equal({
            ...initTreeNode({ id: '1', value: '1' }),
            children: ['1.1'],
            status: TreeNodeStatus.LOADED
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_COLLAPSE - expanded', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1', pickerId: "projects" }))
        )({ projects: createTree() });
        expect(getNode('1')(newState.projects)).to.deep.equal({
            ...initTreeNode({ id: '1', value: '1' }),
            expanded: true
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_COLLAPSE - expanded', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1', pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1', pickerId: "projects" })),
        )({ projects: createTree() });
        expect(getNode('1')(newState.projects)).to.deep.equal({
            ...initTreeNode({ id: '1', value: '1' }),
            expanded: false
        });
    });

    it('ACTIVATE_TREE_PICKER_NODE', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: '1', pickerId: "projects" })),
        )({ projects: createTree() });
        expect(getNode('1')(newState.projects)).to.deep.equal({
            ...initTreeNode({ id: '1', value: '1' }),
            active: true
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_SELECTION', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const subNode = initTreeNode({ id: '1.1', value: '1.1' });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '1', nodes: [subNode], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECTION({ id: '1.1', pickerId: "projects", cascade: true })),
        )({ projects: createTree() });
        expect(getNode('1')(newState.projects)).to.deep.equal({
            ...initTreeNode({ id: '1', value: '1' }),
            selected: true,
            children: ['1.1'],
            status: TreeNodeStatus.LOADED,
        });
    });

    it('does not set malformed node', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const malformedNode = initTreeNode({ id: '', value: NaN });
        const newState = pipe(
            (state) => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: node.id, nodes: [node], pickerId: "projects" })),
            state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: malformedNode.id, nodes: [malformedNode], pickerId: "projects" })),
        )({ projects: createTree() });
        expect(getNode(node.id)(newState.projects)).to.deep.equal({
            active: false,
            children: [ "1" ],
            expanded: false,
            id: "1",
            parent: "1",
            selected: false,
            status: "INITIAL",
            value: "1"
        });
        expect(getNode(malformedNode.id)(newState.projects)).to.equal(undefined);
    });
});
