// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { parseKeepManifestText, parseKeepManifestStream } from "./collection-manifest-parser";

describe('parseKeepManifestText', () => {
    it('should parse text into streams', () => {
        const manifestText = `. 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt\n./c d41d8cd98f00b204e9800998ecf8427e+0 0:0:d\n`;
        const manifest = parseKeepManifestText(manifestText);
        expect(manifest[0].name).toBe('');
        expect(manifest[1].name).toBe('/c');
        expect(manifest.length).toBe(2);
    });
});

describe('parseKeepManifestStream', () => {
    const streamText = './c 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt';
    const stream = parseKeepManifestStream(streamText);

    it('should parse stream name', () => {
        expect(stream.name).toBe('/c');
    });
    it('should parse stream locators', () => {
        expect(stream.locators).toEqual(['930625b054ce894ac40596c3f5a0d947+33']);
    });
    it('should parse stream files', () => {
        expect(stream.files).toEqual([
            {name: 'a', position: '0', size: 0},
            {name: 'b', position: '0', size: 0},
            {name: 'output.txt', position: '0', size: 33},
        ]);
    });
});