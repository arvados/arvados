// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createCollectionFilesTree, CollectionDirectory, CollectionFile, CollectionFileType, createCollectionDirectory, createCollectionFile } from "../../models/collection-file";
import { Tree, mapTree, getNodeChildren, getNode, TreeNode } from "../../models/tree";
import { getTagValue } from "../../common/xml";

export const parseFilesResponse = (document: Document) => {
    const files = extractFilesData(document);
    const tree = createCollectionFilesTree(files);
    return sortFilesTree(tree);
};

export const sortFilesTree = (tree: Tree<CollectionDirectory | CollectionFile>) => {
    return mapTree(node => {
        const children = getNodeChildren(node.id)(tree).map(id => getNode(id)(tree)) as TreeNode<CollectionDirectory | CollectionFile>[];
        children.sort((a, b) =>
            a.value.type !== b.value.type
                ? a.value.type === CollectionFileType.DIRECTORY ? -1 : 1
                : a.value.name.localeCompare(b.value.name)
        );
        return { ...node, children: children.map(child => child.id) } as TreeNode<CollectionDirectory | CollectionFile>;
    })(tree);
};

export const extractFilesData = (document: Document) => {
    const collectionUrlPrefix = /\/c=[0-9a-zA-Z\-]*/;
    return Array
        .from(document.getElementsByTagName('D:response'))
        .slice(1) // omit first element which is collection itself
        .map(element => {
            const name = getTagValue(element, 'D:displayname', '');
            const size = parseInt(getTagValue(element, 'D:getcontentlength', '0'), 10);
            const pathname = getTagValue(element, 'D:href', '');
            const nameSuffix = `/${name || ''}`;
            const directory = pathname
                .replace(collectionUrlPrefix, '')
                .replace(nameSuffix, '');

            const data = {
                url: pathname,
                id: `${directory}/${name}`,
                name,
                path: directory,
            };

            return getTagValue(element, 'D:resourcetype', '')
                ? createCollectionDirectory(data)
                : createCollectionFile({ ...data, size });

        });
};
