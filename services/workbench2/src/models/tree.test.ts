// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as Tree from './tree';
import { initTreeNode } from './tree';
import { pipe } from 'lodash/fp';

describe('Tree', () => {
    let tree: Tree.Tree<string>;

    beforeEach(() => {
        tree = Tree.createTree();
    });

    it('sets new node', () => {
        const newTree = Tree.setNode(initTreeNode({ id: 'Node 1', value: 'Value 1' }))(tree);
        expect(Tree.getNode('Node 1')(newTree)).toEqual(initTreeNode({ id: 'Node 1', value: 'Value 1' }));
    });

    it('appends a subtree', () => {
        const newTree = Tree.setNode(initTreeNode({ id: 'Node 1', value: 'Value 1' }))(tree);
        const subtree = Tree.setNode(initTreeNode({ id: 'Node 2', value: 'Value 2' }))(Tree.createTree());
        const mergedTree = Tree.appendSubtree('Node 1', subtree)(newTree);
        expect(Tree.getNode('Node 1')(mergedTree)).toBeDefined();
        expect(Tree.getNode('Node 2')(mergedTree)).toBeDefined();
    });

    it('adds new node reference to parent children', () => {
        const newTree = pipe(
            Tree.setNode(initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' })),
            Tree.setNode(initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 2' })),
        )(tree);

        expect(Tree.getNode('Node 1')(newTree)).toEqual({
            ...initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            children: ['Node 2']
        });
    });

    it('gets node ancestors', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 2', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeAncestorsIds('Node 3')(newTree)).toEqual(['Node 1', 'Node 2']);
    });

    it('gets node descendants', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeDescendantsIds('Node 1')(newTree)).toEqual(['Node 2', 'Node 3', 'Node 2.1', 'Node 3.1']);
    });

    it('gets root descendants', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeDescendantsIds('')(newTree)).toEqual(['Node 1', 'Node 2', 'Node 3', 'Node 2.1', 'Node 3.1']);
    });

    it('gets node children', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeChildrenIds('Node 1')(newTree)).toEqual(['Node 2', 'Node 3']);
    });

    it('gets root children', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeChildrenIds('')(newTree)).toEqual(['Node 1', 'Node 3']);
    });

    it('maps tree', () => {
        const newTree = [
            initTreeNode({ id: 'Node 1', parent: '', value: 'Value 1' }),
            initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 'Value 2' }),
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        const mappedTree = Tree.mapTreeValues<string, number>(value => parseInt(value.split(' ')[1], 10))(newTree);
        expect(Tree.getNode('Node 2')(mappedTree)).toEqual(initTreeNode({ id: 'Node 2', parent: 'Node 1', value: 2 }));
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

        expect(Tree.getNode('Root Node 1')(expandedTree)?.expanded).toEqual(true);
        expect(Tree.getNode('Node 1.1')(expandedTree)?.expanded).toEqual(true);
        expect(Tree.getNode('Node 1.1.1')(expandedTree)?.expanded).toEqual(false);
        expect(Tree.getNode('Node 1.2')(expandedTree)?.expanded).toEqual(false);
        expect(Tree.getNode('Root Node 2')(expandedTree)?.expanded).toEqual(true);
        expect(Tree.getNode('Node 2.1')(expandedTree)?.expanded).toEqual(false);
        expect(Tree.getNode('Node 2.1.1')(expandedTree)?.expanded).toEqual(false);
        expect(Tree.getNode('Root Node 3')(expandedTree)?.expanded).toEqual(false);
        expect(Tree.getNode('Node 3.1')(expandedTree)?.expanded).toEqual(false);
    });
});
