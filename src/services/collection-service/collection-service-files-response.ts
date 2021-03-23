// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionDirectory, CollectionFile, CollectionFileType, createCollectionDirectory, createCollectionFile } from "../../models/collection-file";
import { getTagValue } from "~/common/xml";
import { getNodeChildren, Tree, mapTree } from '~/models/tree';
import { customDecodeURI } from "~/common/url";

export const sortFilesTree = (tree: Tree<CollectionDirectory | CollectionFile>) => {
    return mapTree<CollectionDirectory | CollectionFile>(node => {
        const children = getNodeChildren(node.id)(tree);

        children.sort((a, b) =>
            a.value.type !== b.value.type
                ? a.value.type === CollectionFileType.DIRECTORY ? -1 : 1
                : a.value.name.localeCompare(b.value.name)
        );
        return { ...node, children: children.map(child => child.id) };
    })(tree);
};

export const extractFilesData = (document: Document) => {
    const collectionUrlPrefix = /\/c=([^\/]*)/;
    return Array
        .from(document.getElementsByTagName('D:response'))
        .slice(1) // omit first element which is collection itself
        .map(element => {
            const name = getTagValue(element, 'D:displayname', '');
            const size = parseInt(getTagValue(element, 'D:getcontentlength', '0'), 10);
            const url = customDecodeURI(getTagValue(element, 'D:href', ''));
            const nameSuffix = name;
            const collectionUuidMatch = collectionUrlPrefix.exec(url);
            const collectionUuid = collectionUuidMatch ? collectionUuidMatch.pop() : '';
            const directory = url
                .replace(collectionUrlPrefix, '')
                .replace(nameSuffix, '')
                .replace(/\/\//g, '/');

            const parentPath = directory.replace(/\/$/, '');
            const data = {
                url,
                id: [
                    collectionUuid ? collectionUuid : '',
                    directory ? parentPath : '',
                    '/' + name
                ].join(''),
                name,
                path: parentPath,
            };

            const result = getTagValue(element, 'D:resourcetype', '')
                ? createCollectionDirectory(data)
                : createCollectionFile({ ...data, size });

            return result;
        });
};

export const getFileFullPath = ({ name, path }: CollectionFile | CollectionDirectory) => {
    return `${path}/${name}`;
};