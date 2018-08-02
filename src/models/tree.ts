import { Children } from "../../node_modules/@types/react";

// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type Tree<T> = Record<string, TreeNode<T>>;

export const TREE_ROOT_ID = '';

export interface TreeNode<T> {
    children: string[];
    value: T;
    id: string;
    parent: string;
}

export const createTree = <T>(): Tree<T> => ({});

export const getNode = (id: string) => <T>(tree: Tree<T>): TreeNode<T> | undefined => tree[id];

export const setNode = <T>(node: TreeNode<T>) => (tree: Tree<T>): Tree<T> => {
    const [newTree] = [tree]
        .map(tree => getNode(node.id)(tree) === node
            ? tree
            : Object.assign({}, tree, { [node.id]: node }))
        .map(addChild(node.parent, node.id));
    return newTree;
};

export const getNodeValue = (id: string) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node ? node.value : undefined;
};

export const setNodeValue = (id: string) => <T>(value: T) => (tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node
        ? setNode(mapNodeValue(() => value)(node))(tree)
        : tree;
};

export const setNodeValueWith = <T>(mapFn: (value: T) => T) => (id: string) => (tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node
        ? setNode(mapNodeValue(mapFn)(node))(tree)
        : tree;
};

export const mapTreeValues = <T, R>(mapFn: (value: T) => R) => (tree: Tree<T>): Tree<R> =>
    getNodeDescendants('')(tree)
        .map(id => getNode(id)(tree))
        .map(mapNodeValue(mapFn))
        .reduce((newTree, node) => setNode(node)(newTree), createTree<R>());

export const mapTree = <T, R>(mapFn: (node: TreeNode<T>) => TreeNode<R>) => (tree: Tree<T>): Tree<R> =>
    getNodeDescendants('')(tree)
        .map(id => getNode(id)(tree))
        .map(mapFn)
        .reduce((newTree, node) => setNode(node)(newTree), createTree<R>());

export const getNodeAncestors = (id: string) => <T>(tree: Tree<T>): string[] => {
    const node = getNode(id)(tree);
    return node && node.parent
        ? [...getNodeAncestors(node.parent)(tree), node.parent]
        : [];
};

export const getNodeDescendants = (id: string, limit = Infinity) => <T>(tree: Tree<T>): string[] => {
    const node = getNode(id)(tree);
    const children = node ? node.children :
        id === TREE_ROOT_ID
            ? getRootNodeChildren(tree)
            : [];

    return children
        .concat(limit < 1
            ? []
            : children
                .map(id => getNodeDescendants(id, limit - 1)(tree))
                .reduce((nodes, nodeChildren) => [...nodes, ...nodeChildren], []));
};

export const getNodeChildren = (id: string) => <T>(tree: Tree<T>): string[] =>
    getNodeDescendants(id, 0)(tree);

const mapNodeValue = <T, R>(mapFn: (value: T) => R) => (node: TreeNode<T>): TreeNode<R> =>
    ({ ...node, value: mapFn(node.value) });

const getRootNodeChildren = <T>(tree: Tree<T>) =>
    Object
        .keys(tree)
        .filter(id => getNode(id)(tree)!.parent === TREE_ROOT_ID);

const addChild = (parentId: string, childId: string) => <T>(tree: Tree<T>): Tree<T> => {
    const node = getNode(parentId)(tree);
    if (node) {
        const children = node.children.some(id => id === childId)
            ? node.children
            : [...node.children, childId];

        const newNode = children === node.children
            ? node
            : { ...node, children };

        return setNode(newNode)(tree);
    }
    return tree;
};
