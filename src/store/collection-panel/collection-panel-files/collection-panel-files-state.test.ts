// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { parseKeepManifestText } from "../../../models/keep-manifest";
import { mapManifestToFiles, mapManifestToDirectories } from './collection-panel-files-state';

test('mapManifestToFiles', () => {
    const manifestText = `. 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt\n./c d41d8cd98f00b204e9800998ecf8427e+0 0:0:d`;
    const manifest = parseKeepManifestText(manifestText);
    const files = mapManifestToFiles(manifest);
    expect(files).toEqual([{
        parentId: '',
        id: '/a',
        name: 'a',
        size: 0,
        selected: false,
        type: 'file'
    }, {
        parentId: '',
        id: '/b',
        name: 'b',
        size: 0,
        selected: false,
        type: 'file'
    }, {
        parentId: '',
        id: '/output.txt',
        name: 'output.txt',
        size: 33,
        selected: false,
        type: 'file'
    }, {
        parentId: '/c',
        id: '/c/d',
        name: 'd',
        size: 0,
        selected: false,
        type: 'file'
    },]);
});

test('mapManifestToDirectories', () => {
    const manifestText = `./c/user/results 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt\n`;
    const manifest = parseKeepManifestText(manifestText);
    const directories = mapManifestToDirectories(manifest);
    expect(directories).toEqual([{
        parentId: "",
        id: '/c',
        name: 'c',
        collapsed: true,
        selected: false,
        type: 'directory'
    }, {
        parentId: '/c',
        id: '/c/user',
        name: 'user',
        collapsed: true,
        selected: false,
        type: 'directory'
    }, {
        parentId: '/c/user',
        id: '/c/user/results',
        name: 'results',
        collapsed: true,
        selected: false,
        type: 'directory'
    },]);
});