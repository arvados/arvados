// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { uniqBy, flow, groupBy } from 'lodash';
import { KeepManifestStream, KeepManifestStreamFile, KeepManifest } from "../../models/keep-manifest";
import { TreeNode, setNode, createTree, getNodeDescendants, getNodeValue, getNode } from '../../models/tree';
import { CollectionFilesTree, CollectionFile, CollectionDirectory, createCollectionDirectory, createCollectionFile, CollectionFileType } from '../../models/collection-file';

export const mapCollectionFilesTreeToManifest = (tree: CollectionFilesTree): KeepManifest => {
    const values = getNodeDescendants('')(tree).map(id => getNodeValue(id)(tree));
    const files = values.filter(value => value && value.type === CollectionFileType.FILE) as CollectionFile[];
    const fileGroups = groupBy(files, file => file.path);
    return Object
        .keys(fileGroups)
        .map(dirName => ({
            name: dirName,
            locators: [],
            files: fileGroups[dirName].map(mapCollectionFile)
        }));
};

export const mapManifestToCollectionFilesTree = (manifest: KeepManifest): CollectionFilesTree =>
    manifestToCollectionFiles(manifest)
        .map(mapCollectionFileToTreeNode)
        .reduce((tree, node) => setNode(node)(tree), createTree<CollectionFile>());


export const mapCollectionFileToTreeNode = (file: CollectionFile): TreeNode<CollectionFile> => ({
    children: [],
    id: file.id,
    parent: file.path,
    value: file
});

export const manifestToCollectionFiles = (manifest: KeepManifest): Array<CollectionDirectory | CollectionFile> => ([
    ...mapManifestToDirectories(manifest),
    ...mapManifestToFiles(manifest)
]);

export const mapManifestToDirectories = (manifest: KeepManifest): CollectionDirectory[] =>
    uniqBy(
        manifest
            .map(mapStreamDirectory)
            .map(splitDirectory)
            .reduce((all, splitted) => ([...all, ...splitted]), []),
        directory => directory.id);

export const mapManifestToFiles = (manifest: KeepManifest): CollectionFile[] =>
    manifest
        .map(stream => stream.files.map(mapStreamFile(stream)))
        .reduce((all, current) => ([...all, ...current]), []);

const splitDirectory = (directory: CollectionDirectory): CollectionDirectory[] => {
    return directory.name
        .split('/')
        .slice(1)
        .map(mapPathComponentToDirectory);
};

const mapPathComponentToDirectory = (component: string, index: number, components: string[]): CollectionDirectory =>
    createCollectionDirectory({
        path: index === 0 ? '' : joinPathComponents(components, index),
        id: joinPathComponents(components, index + 1),
        name: component,
    });

const joinPathComponents = (components: string[], index: number) =>
    `/${components.slice(0, index).join('/')}`;

const mapCollectionFile = (file: CollectionFile): KeepManifestStreamFile => ({
    name: file.name,
    position: '',
    size: file.size
});

const mapStreamDirectory = (stream: KeepManifestStream): CollectionDirectory =>
    createCollectionDirectory({
        path: '',
        id: stream.name,
        name: stream.name,
    });

const mapStreamFile = (stream: KeepManifestStream) =>
    (file: KeepManifestStreamFile): CollectionFile =>
        createCollectionFile({
            path: stream.name,
            id: `${stream.name}/${file.name}`,
            name: file.name,
            size: file.size,
        });

