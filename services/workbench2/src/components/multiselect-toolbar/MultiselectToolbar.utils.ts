// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TCheckedList } from "components/data-table/data-table";
import { extractUuidKind } from "models/resource";
import { isUserGroup } from "models/group";
import { getResource, ResourcesState } from "store/resources/resources";
import { ContextMenuKind } from 'store/context-menu/context-menu';

const detailsCardPaths = [
    '/projects',
    '/workflows',
    '/collections',
    '/processes',
];

export const usesDetailsCard = (location: string): boolean => {
    return detailsCardPaths.some(path => location.includes(path));
};

export function selectedToArray(checkedList: TCheckedList): Array<string> {
    const arrayifiedSelectedList: Array<string> = [];
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            arrayifiedSelectedList.push(key);
        }
    }
    return arrayifiedSelectedList;
}

export function selectedToKindSet(checkedList: TCheckedList, resources: ResourcesState = {}): Set<string> {
    const setifiedList = new Set<string>();
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            isRoleGroupResource(key, resources) ? setifiedList.add(ContextMenuKind.GROUPS) : setifiedList.add(extractUuidKind(key) as string);
        }
    }
    return setifiedList;
}

export const isRoleGroupResource = (uuid: string, resources: ResourcesState): boolean => {
    const resource = getResource(uuid)(resources);
    return isUserGroup(resource);
};
