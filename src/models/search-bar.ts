// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from '~/models/resource';

export type SearchBarAdvanceFormData = {
    type?: ResourceKind;
    cluster?: ClusterObjectType;
    projectUuid?: string;
    inTrash: boolean;
    dateFrom: string;
    dateTo: string;
    saveQuery: boolean;
    queryName: string;
    searchValue: string;
    properties: PropertyValues[];
};

export interface PropertyValues {
    key: string;
    value: string;
}

export enum ClusterObjectType {
    INDIANAPOLIS = "indianapolis",
    KAISERAUGST = "kaiseraugst",
    PENZBERG = "penzberg"
}

export const getClusterObjectType = (cluster?: string) => {
    switch (cluster) {
        case "indianapolis":
            return ClusterObjectType.INDIANAPOLIS;
        case "kaiseraugst":
            return ClusterObjectType.KAISERAUGST;
        case "penzberg":
            return ClusterObjectType.PENZBERG;
        default:
            return undefined;
    }
};
