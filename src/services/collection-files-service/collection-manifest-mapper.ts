// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { uniqBy } from 'lodash';
import { KeepManifestStream, KeepManifestStreamFile, KeepManifest } from "../../models/keep-manifest";
import { TreeNode, setNode, createTree } from '../../models/tree';
import { CollectionFilesTree, CollectionFile, CollectionDirectory, createCollectionDirectory, createCollectionFile } from '../../models/collection-file';

export const mapManifestToCollectionFilesTree = (manifest: KeepManifest): CollectionFilesTree =>
    manifestToCollectionFiles(manifest)
        .map(mapCollectionFileToTreeNode)
        .reduce((tree, node) => setNode(node)(tree), createTree<CollectionFile>());


export const mapCollectionFileToTreeNode = (file: CollectionFile): TreeNode<CollectionFile> => ({
    children: [],
    id: file.id,
    parent: file.parentId,
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
        parentId: index === 0 ? '' : joinPathComponents(components, index),
        id: joinPathComponents(components, index + 1),
        name: component,
    });

const joinPathComponents = (components: string[], index: number) =>
    `/${components.slice(0, index).join('/')}`;

const mapStreamDirectory = (stream: KeepManifestStream): CollectionDirectory =>
    createCollectionDirectory({
        parentId: '',
        id: stream.name,
        name: stream.name,
    });

const mapStreamFile = (stream: KeepManifestStream) =>
    (file: KeepManifestStreamFile): CollectionFile =>
        createCollectionFile({
            parentId: stream.name,
            id: `${stream.name}/${file.name}`,
            name: file.name,
            size: file.size,
        });

