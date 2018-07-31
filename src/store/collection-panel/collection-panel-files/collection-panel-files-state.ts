// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { uniqBy } from 'lodash';
import { KeepManifestStream, KeepManifestStreamFile, KeepManifest } from "../../../models/keep-manifest";

export type CollectionPanelFilesState = Array<CollectionPanelItem>;

export type CollectionPanelItem = CollectionPanelDirectory | CollectionPanelFile;

export interface CollectionPanelDirectory {
    parentId?: string;
    id: string;
    name: string;
    collapsed: boolean;
    selected: boolean;
    type: 'directory';
}

export interface CollectionPanelFile {
    parentId?: string;
    id: string;
    name: string;
    selected: boolean;
    size: number;
    type: 'file';
}

export const mapManifestToItems = (manifest: KeepManifest): CollectionPanelItem[] => ([
    ...mapManifestToDirectories(manifest),
    ...mapManifestToFiles(manifest)
]);

export const mapManifestToDirectories = (manifest: KeepManifest): CollectionPanelDirectory[] =>
    uniqBy(
        manifest
            .map(mapStreamDirectory)
            .map(splitDirectory)
            .reduce((all, splitted) => ([...all, ...splitted]), []),
        directory => directory.id);

export const mapManifestToFiles = (manifest: KeepManifest): CollectionPanelFile[] =>
    manifest
        .map(stream => stream.files.map(mapStreamFile(stream)))
        .reduce((all, current) => ([...all, ...current]), []);

const splitDirectory = (directory: CollectionPanelDirectory): CollectionPanelDirectory[] => {
    return directory.name
        .split('/')
        .slice(1)
        .map(mapPathComponentToDirectory);
};

const mapPathComponentToDirectory = (component: string, index: number, components: string[]): CollectionPanelDirectory =>
    createDirectory({
        parentId: index === 0 ? '' : joinPathComponents(components, index),
        id: joinPathComponents(components, index + 1),
        name: component,
    });

const joinPathComponents = (components: string[], index: number) =>
    `/${components.slice(0, index).join('/')}`;

const mapStreamDirectory = (stream: KeepManifestStream): CollectionPanelDirectory =>
    createDirectory({
        parentId: '',
        id: stream.name,
        name: stream.name,
    });

const mapStreamFile = (stream: KeepManifestStream) =>
    (file: KeepManifestStreamFile): CollectionPanelFile =>
        createFile({
            parentId: stream.name,
            id: `${stream.name}/${file.name}`,
            name: file.name,
            size: file.size,
        });

const createDirectory = (data: { parentId: string, id: string, name: string }): CollectionPanelDirectory => ({
    ...data,
    collapsed: true,
    selected: false,
    type: 'directory'
});

const createFile = (data: { parentId: string, id: string, name: string, size: number }): CollectionPanelFile => ({
    ...data,
    selected: false,
    type: 'file'
});