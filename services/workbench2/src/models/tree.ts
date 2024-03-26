// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { pipe, map, reduce } from 'lodash/fp';
export type Tree<T> = Record<string, TreeNode<T>>;

export const TREE_ROOT_ID = '';

export interface TreeNode<T = any> {
    children: string[];
    value: T;
    id: string;
    parent: string;
    active: boolean;
    selected: boolean;
    initialState?: boolean;
    expanded: boolean;
    status: TreeNodeStatus;
}

export enum TreeNodeStatus {
    INITIAL = 'INITIAL',
    PENDING = 'PENDING',
    LOADED = 'LOADED',
}

export enum TreePickerId {
    PROJECTS = 'Projects',
    SHARED_WITH_ME = 'Shared with me',
    FAVORITES = 'Favorites',
    PUBLIC_FAVORITES = 'Public Favorites'
}

export const createTree = <T>(): Tree<T> => ({});

export const getNode = (id: string) => <T>(tree: Tree<T>): TreeNode<T> | undefined => tree[id];

export const appendSubtree = <T>(id: string, subtree: Tree<T>) => (tree: Tree<T>) =>
    pipe(
        getNodeDescendants(''),
        map(node => node.parent === '' ? { ...node, parent: id } : node),
        reduce((newTree, node) => setNode(node)(newTree), tree)
    )(subtree) as Tree<T>;

export const setNode = <T>(node: TreeNode<T>) => (tree: Tree<T>): Tree<T> => {
    if (tree[node.id] && tree[node.id] === node) { return tree; }

    tree[node.id] = node;
    if (tree[node.parent]) {
        tree[node.parent].children = Array.from(new Set([...tree[node.parent].children, node.id]));
    }
    return tree;
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
        .filter(node => !!node)
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

export const countNodes = <T>(tree: Tree<T>) =>
    getNodeDescendantsIds('')(tree).length;

export const countChildren = (id: string) => <T>(tree: Tree<T>) =>
    getNodeChildren('')(tree).length;

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
    mapTree((node: TreeNode<T>) => node.id === id ? { ...node, active: true } : { ...node, active: false })(tree);

export const deactivateNode = <T>(tree: Tree<T>) =>
    mapTree((node: TreeNode<T>) => node.active ? { ...node, active: false } : node)(tree);

export const expandNode = (...ids: string[]) => <T>(tree: Tree<T>) =>
    mapTree((node: TreeNode<T>) => ids.some(id => id === node.id) ? { ...node, expanded: true } : node)(tree);

export const expandNodeAncestors = (...ids: string[]) => <T>(tree: Tree<T>) => {
    const ancestors = ids.reduce((acc, id): string[] => ([...acc, ...getNodeAncestorsIds(id)(tree)]), [] as string[]);
    return mapTree((node: TreeNode<T>) => ancestors.some(id => id === node.id) ? { ...node, expanded: true } : node)(tree);
}

export const collapseNode = (...ids: string[]) => <T>(tree: Tree<T>) =>
    mapTree((node: TreeNode<T>) => ids.some(id => id === node.id) ? { ...node, expanded: false } : node)(tree);

export const toggleNodeCollapse = (...ids: string[]) => <T>(tree: Tree<T>) =>
    mapTree((node: TreeNode<T>) => ids.some(id => id === node.id) ? { ...node, expanded: !node.expanded } : node)(tree);

export const setNodeStatus = (id: string) => (status: TreeNodeStatus) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node
        ? setNode({ ...node, status })(tree)
        : tree;
};

export const toggleNodeSelection = (id: string, cascade: boolean) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);

    return node
        ? cascade
            ? pipe(
                setNode({ ...node, selected: !node.selected }),
                toggleAncestorsSelection(id),
                toggleDescendantsSelection(id))(tree)
            : setNode({ ...node, selected: !node.selected })(tree)
        : tree;
};

export const selectNode = (id: string, cascade: boolean) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node && node.selected
        ? tree
        : toggleNodeSelection(id, cascade)(tree);
};

export const selectNodes = (id: string | string[], cascade: boolean) => <T>(tree: Tree<T>) => {
    const ids = typeof id === 'string' ? [id] : id;
    return ids.reduce((tree, id) => selectNode(id, cascade)(tree), tree);
};
export const deselectNode = (id: string, cascade: boolean) => <T>(tree: Tree<T>) => {
    const node = getNode(id)(tree);
    return node && node.selected
        ? toggleNodeSelection(id, cascade)(tree)
        : tree;
};

export const deselectNodes = (id: string | string[], cascade: boolean) => <T>(tree: Tree<T>) => {
    const ids = typeof id === 'string' ? [id] : id;
    return ids.reduce((tree, id) => deselectNode(id, cascade)(tree), tree);
};

export const getSelectedNodes = <T>(tree: Tree<T>) =>
    getNodeDescendants('')(tree)
        .filter(node => node.selected);

export const initTreeNode = <T>(data: Pick<TreeNode<T>, 'id' | 'value'> & { parent?: string }): TreeNode<T> => ({
    children: [],
    active: false,
    selected: false,
    expanded: false,
    status: TreeNodeStatus.INITIAL,
    parent: '',
    ...data,
});

export const getTreeDirty = (id: string) => <T>(tree: Tree<T>): boolean => {
    const node = getNode(id)(tree);
    const children = getNodeDescendants(id)(tree);
    return (node
            && node.initialState !== undefined
            && node.selected !== node.initialState
            )
            || children.some(child =>
                child.initialState !== undefined
                && child.selected !== child.initialState
            );
}

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
