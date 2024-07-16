// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as Tree from './tree';
import { initTreeNode } from './tree';
import { pipe, isEqual } from 'lodash/fp';

describe('Tree', () => {
    let tree;

    beforeEach(() => {
        tree = Tree.createTree();
    });

    it('sets new node', () => {
        const newTree = Tree.setNode(initTreeNode({ id: 'Node 1', value: 'Value 1' }))(tree);
        expect(isEqual(Tree.getNode('Node 1')(newTree), initTreeNode({ id: 'Node 1', value: 'Value 1' }))).to.equal(true);
    });

    it('appends a subtree', () => {
        const newTree = Tree.setNode(initTreeNode({ id: 'Node 1', value: 'Value 1' }))(tree);
        const subtree = Tree.setNode(initTreeNode({ id: 'Node 2', value: 'Value 2' }))(Tree.createTree());
        const mergedTree = Tree.appendSubtree('Node 1', subtree)(newTree);
        expect(Tree.getNode('Node 1')(mergedTree)).to.not.equal('undefined');
        expect(Tree.getNode('Node 2')(mergedTree)).to.not.equal('undefined');
    });

    it('adds new node reference to parent children', () => {
        const newTree = pipe(
            Tree.setNode(initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' })),
            Tree.setNode(initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 2' })),
        )(tree);

        expect(isEqual(Tree.getNode('Node 1')(newTree), {
            ...initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            children: ['Node 2']
        })).to.equal(true);
    });

    it('gets node ancestors', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 2', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(isEqual(Tree.getNodeAncestorsIds('Node 3')(newTree), ['Node 1', 'Node 2'])).to.equal(true);
    });

    it('gets node descendants', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(isEqual(Tree.getNodeDescendantsIds('Node 1')(newTree), ['Node 2', 'Node 3', 'Node 2.1', 'Node 3.1'])).to.equal(true);
    });

    it('gets root descendants', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(isEqual(Tree.getNodeDescendantsIds('')(newTree), ['Node 1', 'Node 2', 'Node 3', 'Node 2.1', 'Node 3.1'])).to.equal(true);
    });

    it('gets node children', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(isEqual(Tree.getNodeChildrenIds('Node 1')(newTree), ['Node 2', 'Node 3'])).to.equal(true);
    });

    it('gets root children', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(isEqual(Tree.getNodeChildrenIds('')(newTree), ['Node 1', 'Node 3'])).to.equal(true);
    });

    it('maps tree', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 2' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        const mappedTree = Tree.mapTreeValues(value => parseInt(value.split(' ')[1], 10))(newTree);
        expect(isEqual(Tree.getNode('Node 2')(mappedTree), initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 2 }))).to.equal(true);
    });

    it('expands node ancestor chains', () => {
        const newTree = [
            initTreeNode({ id: 'Root Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 1.1', parent: 'Root Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 1.1.1', parent: 'Node 1.1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 1.2', parent: 'Root Node 1', value: 'Value 1' }),

            initTreeNode({ id: 'Root Node 2', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Root Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1.1', parent: 'Node 2.1', value: 'Value 1' }),

            initTreeNode({ id: 'Root Node 3', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Root Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);

        const expandedTree = Tree.expandNodeAncestors(
            'Node 1.1.1', // Expands 1.1 and 1
            'Node 2.1', // Expands 2
        )(newTree);

        expect(Tree.getNode('Root Node 1')(expandedTree)?.expanded).to.equal(true);
        expect(Tree.getNode('Node 1.1')(expandedTree)?.expanded).to.equal(true);
        expect(Tree.getNode('Node 1.1.1')(expandedTree)?.expanded).to.equal(false);
        expect(Tree.getNode('Node 1.2')(expandedTree)?.expanded).to.equal(false);
        expect(Tree.getNode('Root Node 2')(expandedTree)?.expanded).to.equal(true);
        expect(Tree.getNode('Node 2.1')(expandedTree)?.expanded).to.equal(false);
        expect(Tree.getNode('Node 2.1.1')(expandedTree)?.expanded).to.equal(false);
        expect(Tree.getNode('Root Node 3')(expandedTree)?.expanded).to.equal(false);
        expect(Tree.getNode('Node 3.1')(expandedTree)?.expanded).to.equal(false);
    });
});
