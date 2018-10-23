// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { pipe } from 'lodash/fp';
export type Tree<T> = Record<string, TreeNode<T>>;

export const TREE_ROOT_ID = '';

export interface TreeNode<T = any> {
    children: string[];
    value: T;
    id: string;
    parent: string;
    active: boolean;
    selected: boolean;
    expanded: boolean;
    status: TreeNodeStatus;
}

export enum TreeNodeStatus {
    INITIAL = 'INITIAL',
    PENDING = 'PENDING',
    LOADED = 'LOADED',
}

export const createTree = <T>(): Tree<T> => ({});

export const getNode = (id: string) => <T>(tree: Tree<T>): TreeNode<T> | undefined => tree[id];

export const setNode = <T>(node: TreeNode<T>) => (tree: Tree<T>): Tree<T> => {
    return pipe(
        (tree: Tree<T>) => getNode(node.id)(tree) === node
            ? tree
            : { ...tree, [node.id]: node },
        addChild(node.parent, node.id)
    )(tree);
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
    getNodeDescendantsIds('')(tree)
        .map(id => getNode(id)(tree))
        .map(mapNodeValue(mapFn))
        .reduce((newTree, node) => setNode(node)(newTree), createTree<R>());

export const mapTree = <T, R = T>(mapFn: (node: TreeNode<T>) => TreeNode<R>) => (tree: Tree<T>): Tree<R> =>
    getNodeDescendantsIds('')(tree)
        .map(id => getNode(id)(tree))
        .map(mapFn)
        .reduce((newTree, node) => setNode(node)(newTree), createTree<R>());

export const getNodeAncestors = (id: string) => <T>(tree: Tree<T>) =>
    mapIdsToNodes(getNodeAncestorsIds(id)(tree))(tree);


export const getNodeAncestorsIds = (id: string) => <T>(tree: Tree<T>): string[] => {
    const node = getNode(id)(tree);
    return node && node.parent
        ? [...getNodeAncestorsIds(node.parent)(tree), node.parent]
        : [];
};

export const getNodeDescendants = (id: string, limit = Infinity) => <T>(tree: Tree<T>) =>
    mapIdsToNodes(getNodeDescendantsIds(id, limit)(tree))(tree);

export const getNodeDescendantsIds = (id: string, limit = Infinity) => <T>(tree: Tree<T>): string[] => {
    const node = getNode(id)(tree);
    const children = node ? node.children :
        id === TREE_ROOT_ID
            ? getRootNodeChildrenIds(tree)
            : [];

    return children
        .concat(limit < 1
            ? []
            : children
                .map(id => getNodeDescendantsIds(id, limit - 1)(tree))
                .reduce((nodes, nodeChildren) => [...nodes, ...nodeChildren], []));
};

export const getNodeChildren = (id: string) => <T>(tree: Tree<T>) =>
    mapIdsToNodes(getNodeChildrenIds(id)(tree))(tree);

export const getNodeChildrenIds = (id: string) => <T>(tree: Tree<T>): string[] =>
    getNodeDescendantsIds(id, 0)(tree);

export const mapIdsToNodes = (ids: string[]) => <T>(tree: Tree<T>) =>
    ids.map(id => getNode(id)(tree)).filter((node): node is TreeNode<T> => node !== undefined);

export const activateNode = (id: string) => <T>(tree: Tree<T>) =>
    mapTree(node => node.id === id ? { ...node, active: true } : { ...node, active: false })(tree);

export const deactivateNode = <T>(tree: Tree<T>) =>
    mapTree(node => node.active ? { ...node, active: false } : node)(tree);

export const expandNode = (...ids: string[]) => <T>(tree: Tree<T>) =>
    mapTree(node => ids.some(id => id === node.id) ? { ...node, expanded: true } : node)(tree);

export const collapseNode = (...ids: string[]) => <T>(tree: Tree<T>) =>
    mapTree(node => ids.some(id => id === node.id) ? { ...node, expanded: false } : node)(tree);

export const toggleNodeCollapse = (...ids: string[]) => <T>(tree: Tree<T>) =>
    mapTree(node => ids.some(id => id === node.id) ? { ...node, expanded: !node.expanded } : node)(tree);

export const setNodeStatus = (id: string) => (status: TreeNodeStatus) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node
        ? setNode({ ...node, status })(tree)
        : tree;
};

export const toggleNodeSelection = (id: string) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node
        ? pipe(
            setNode({ ...node, selected: !node.selected }),
            toggleAncestorsSelection(id),
            toggleDescendantsSelection(id))(tree)
        : tree;

};

export const selectNode = (id: string) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node && node.selected
        ? tree
        : toggleNodeSelection(id)(tree);
};

export const selectNodes = (id: string | string[]) => <T>(tree: Tree<T>) => {
    const ids = typeof id === 'string' ? [id] : id;
    return ids.reduce((tree, id) => selectNode(id)(tree), tree);
};
export const deselectNode = (id: string) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node && node.selected
        ? toggleNodeSelection(id)(tree)
        : tree;
};

export const deselectNodes = (id: string | string[]) => <T>(tree: Tree<T>) => {
    const ids = typeof id === 'string' ? [id] : id;
    return ids.reduce((tree, id) => deselectNode(id)(tree), tree);
};

export const initTreeNode = <T>(data: Pick<TreeNode<T>, 'id' | 'value'> & { parent?: string }): TreeNode<T> => ({
    children: [],
    active: false,
    selected: false,
    expanded: false,
    status: TreeNodeStatus.INITIAL,
    parent: '',
    ...data,
});

const toggleDescendantsSelection = (id: string) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    if (node) {
        return getNodeDescendants(id)(tree)
            .reduce((newTree, subNode) =>
                setNode({ ...subNode, selected: node.selected })(newTree),
                tree);
    }
    return tree;
};

const toggleAncestorsSelection = (id: string) => <T>(tree: Tree<T>) => {
    const ancestors = getNodeAncestorsIds(id)(tree).reverse();
    return ancestors.reduce((newTree, parent) => parent ? toggleParentNodeSelection(parent)(newTree) : newTree, tree);
};

const toggleParentNodeSelection = (id: string) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    if (node) {
        const parentNode = getNode(node.id)(tree);
        if (parentNode) {
            const selected = parentNode.children
                .map(id => getNode(id)(tree))
                .every(node => node !== undefined && node.selected);
            return setNode({ ...parentNode, selected })(tree);
        }
        return setNode(node)(tree);
    }
    return tree;
};


const mapNodeValue = <T, R>(mapFn: (value: T) => R) => (node: TreeNode<T>): TreeNode<R> =>
    ({ ...node, value: mapFn(node.value) });

const getRootNodeChildrenIds = <T>(tree: Tree<T>) =>
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
