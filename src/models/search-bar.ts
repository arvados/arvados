// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from '~/models/resource';

export interface SearchBarAdvanceFormData {
    type?: ResourceKind;
    cluster?: ClusterObjectType;
    project?: string;
    inTrash: boolean;
    dateFrom: string;
    dateTo: string;
    saveQuery: boolean;
    searchQuery: string;
}

export enum ClusterObjectType {
    INDIANAPOLIS = "indianapolis",
    KAISERAUGST = "kaiseraugst",
    PENZBERG = "penzberg"
}