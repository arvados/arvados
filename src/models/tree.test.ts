// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as Tree from './tree';

describe('Tree', () => {
    let tree: Tree.Tree<string>;

    beforeEach(() => {
        tree = Tree.createTree();
    });

    it('sets new node', () => {
        const newTree = Tree.setNode({ children: [], id: 'Node 1', parent: '', value: 'Value 1' })(tree);
        expect(Tree.getNode('Node 1')(newTree)).toEqual({ children: [], id: 'Node 1', parent: '', value: 'Value 1' });
    });

    it('adds new node reference to parent children', () => {
        const [newTree] = [tree]
            .map(Tree.setNode({ children: [], id: 'Node 1', parent: '', value: 'Value 1' }))
            .map(Tree.setNode({ children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 2' }));

        expect(Tree.getNode('Node 1')(newTree)).toEqual({ children: ['Node 2'], id: 'Node 1', parent: '', value: 'Value 1' });
    });

    it('gets node ancestors', () => {
        const newTree = [
            { children: [], id: 'Node 1', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 3', parent: 'Node 2', value: 'Value 1' }
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeAncestors('Node 3')(newTree)).toEqual(['Node 1', 'Node 2']);
    });

    it('gets node descendants', () => {
        const newTree = [
            { children: [], id: 'Node 1', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' },
            { children: [], id: 'Node 3', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeDescendants('Node 1')(newTree)).toEqual(['Node 2', 'Node 3', 'Node 2.1', 'Node 3.1']);
    });

    it('gets root descendants', () => {
        const newTree = [
            { children: [], id: 'Node 1', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' },
            { children: [], id: 'Node 3', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeDescendants('')(newTree)).toEqual(['Node 1', 'Node 2', 'Node 3', 'Node 2.1', 'Node 3.1']);
    });

    it('gets node children', () => {
        const newTree = [
            { children: [], id: 'Node 1', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' },
            { children: [], id: 'Node 3', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeChildren('Node 1')(newTree)).toEqual(['Node 2', 'Node 3']);
    });

    it('gets root children', () => {
        const newTree = [
            { children: [], id: 'Node 1', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 1' },
            { children: [], id: 'Node 2.1', parent: 'Node 2', value: 'Value 1' },
            { children: [], id: 'Node 3', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 3.1', parent: 'Node 3', value: 'Value 1' }
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        expect(Tree.getNodeChildren('')(newTree)).toEqual(['Node 1', 'Node 3']);
    });

    it('maps tree', () => {
        const newTree = [
            { children: [], id: 'Node 1', parent: '', value: 'Value 1' },
            { children: [], id: 'Node 2', parent: 'Node 1', value: 'Value 2' },
        ].reduce((tree, node) => Tree.setNode(node)(tree), tree);
        const mappedTree = Tree.mapTreeValues<string, number>(value => parseInt(value.split(' ')[1], 10))(newTree);
        expect(Tree.getNode('Node 2')(mappedTree)).toEqual({ children: [], id: 'Node 2', parent: 'Node 1', value: 2 }, );
    });
});