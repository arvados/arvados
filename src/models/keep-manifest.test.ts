// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { parseKeepManifestText, parseKeepManifestStream } from "./keep-manifest";

describe('parseKeepManifestText', () => {
    it('should return correct number of streams', () => {
        const manifestText = `. 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt
        ./c d41d8cd98f00b204e9800998ecf8427e+0 0:0:d`;
        const manifest = parseKeepManifestText(manifestText);
        expect(manifest).toHaveLength(2);
    });
});

describe('parseKeepManifestStream', () => {
    const streamText = './c 930625b054ce894ac40596c3f5a0d947+33 0:0:a 0:0:b 0:33:output.txt';
    const stream = parseKeepManifestStream(streamText);

    it('should parse stream name', () => {
        expect(stream.streamName).toBe('./c');
    });
    it('should parse stream locators', () => {
        expect(stream.locators).toEqual(['930625b054ce894ac40596c3f5a0d947+33']);
    });
    it('should parse stream files', () => {
        expect(stream.files).toEqual([
            {fileName: 'a', position: '0', size: 0},
            {fileName: 'b', position: '0', size: 0},
            {fileName: 'output.txt', position: '0', size: 33},
        ]);
    });
});