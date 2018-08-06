// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { parseKeepManifestText } from "./collection-manifest-parser";
import { mapManifestToFiles, mapManifestToDirectories, mapManifestToCollectionFilesTree, mapCollectionFilesTreeToManifest } from "./collection-manifest-mapper";

test('mapManifestToFiles', () => {
    const manifestText = `. 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt\n./c d41d8cd98f00b204e9800998ecf8427e+0 0:0:d`;
    const manifest = parseKeepManifestText(manifestText);
    const files = mapManifestToFiles(manifest);
    expect(files).toEqual([{
        path: '',
        id: '/a',
        name: 'a',
        size: 0,
        type: 'file'
    }, {
        path: '',
        id: '/b',
        name: 'b',
        size: 0,
        type: 'file'
    }, {
        path: '',
        id: '/output.txt',
        name: 'output.txt',
        size: 33,
        type: 'file'
    }, {
        path: '/c',
        id: '/c/d',
        name: 'd',
        size: 0,
        type: 'file'
    },]);
});

test('mapManifestToDirectories', () => {
    const manifestText = `./c/user/results 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt\n`;
    const manifest = parseKeepManifestText(manifestText);
    const directories = mapManifestToDirectories(manifest);
    expect(directories).toEqual([{
        path: "",
        id: '/c',
        name: 'c',
        type: 'directory'
    }, {
        path: '/c',
        id: '/c/user',
        name: 'user',
        type: 'directory'
    }, {
        path: '/c/user',
        id: '/c/user/results',
        name: 'results',
        type: 'directory'
    },]);
});

test('mapCollectionFilesTreeToManifest', () => {
    const manifestText = `. 930625b054ce894ac40596c3f5a0d947+33 0:22:test.txt\n./c/user/results 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt\n`;
    const tree = mapManifestToCollectionFilesTree(parseKeepManifestText(manifestText));
    const manifest = mapCollectionFilesTreeToManifest(tree);
    expect(manifest).toEqual([{
        name: '',
        locators: [],
        files: [{
            name: 'test.txt',
            position: '',
            size: 22
        },],
    }, {
        name: '/c/user/results',
        locators: [],
        files: [{
            name: 'a',
            position: '',
            size: 0
        }, {
            name: 'b',
            position: '',
            size: 0
        }, {
            name: 'output.txt',
            position: '',
            size: 33
        },],
    },]);

});