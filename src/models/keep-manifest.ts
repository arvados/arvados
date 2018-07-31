// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type KeepManifest = KeepManifestStream[];

export interface KeepManifestStream {
    name: string;
    locators: string[];
    files: Array<KeepManifestStreamFile>;
}

export interface KeepManifestStreamFile {
    name: string;
    position: string;
    size: number;
}

/**
 * Documentation [http://doc.arvados.org/api/storage.html](http://doc.arvados.org/api/storage.html)
 */
export const parseKeepManifestText = (text: string) =>
    text
        .split(/\n/)
        .filter(streamText => streamText.length > 0)
        .map(parseKeepManifestStream);

/**
 * Documentation [http://doc.arvados.org/api/storage.html](http://doc.arvados.org/api/storage.html)
 */
export const parseKeepManifestStream = (stream: string): KeepManifestStream => {
    const tokens = stream.split(' ');
    return {
        name: streamName(tokens),
        locators: locators(tokens),
        files: files(tokens)
    };
};

const FILE_LOCATOR_REGEXP = /^([0-9a-f]{32})\+([0-9]+)(\+[A-Z][-A-Za-z0-9@_]*)*$/;

const FILE_REGEXP = /([0-9]+):([0-9]+):(.*)/;

const streamName = (tokens: string[]) => tokens[0].slice(1);

const locators = (tokens: string[]) => tokens.filter(isFileLocator);

const files = (tokens: string[]) => tokens.filter(isFile).map(parseFile);

const isFileLocator = (token: string) => FILE_LOCATOR_REGEXP.test(token);

const isFile = (token: string) => FILE_REGEXP.test(token);

const parseFile = (token: string): KeepManifestStreamFile => {
    const match = FILE_REGEXP.exec(token);
    const [position, size, name] = match!.slice(1);
    return { name, position, size: parseInt(size, 10) };
};