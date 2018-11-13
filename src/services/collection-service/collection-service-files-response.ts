// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createCollectionFilesTree, CollectionDirectory, CollectionFile, CollectionFileType, createCollectionDirectory, createCollectionFile } from "../../models/collection-file";
import { getTagValue } from "~/common/xml";
import { getNodeChildren, Tree, mapTree } from '~/models/tree';

export const parseFilesResponse = (document: Document) => {
    const files = extractFilesData(document);
    const tree = createCollectionFilesTree(files);
    return sortFilesTree(tree);
};

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
            const url = getTagValue(element, 'D:href', '');
            const nameSuffix = `/${name || ''}`;
            const collectionUuidMatch = collectionUrlPrefix.exec(url);
            const collectionUuid = collectionUuidMatch ? collectionUuidMatch.pop() : '';
            const directory = url
                .replace(collectionUrlPrefix, '')
                .replace(nameSuffix, '');


            const data = {
                url,
                id: [
                    collectionUuid ? collectionUuid : '',
                    directory ? '/' + directory.replace(/^\//, '') : '',
                    '/' + name
                ].join(''),
                name,
                path: directory,
            };

            return getTagValue(element, 'D:resourcetype', '')
                ? createCollectionDirectory(data)
                : createCollectionFile({ ...data, size });

        });
};

export const getFileFullPath = ({ name, path }: CollectionFile | CollectionDirectory) =>
    `${path}/${name}`;
