// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from 'models/resource';

export type SearchBarAdvancedFormData = {
    type?: ResourceKind;
    cluster?: string;
    projectUuid?: string;
    inTrash: boolean;
    pastVersions: boolean;
    dateFrom: string;
    dateTo: string;
    saveQuery: boolean;
    queryName: string;
    searchValue: string;
    properties: PropertyValue[];
};

export interface PropertyValue {
    key: string;
    keyID?: string;
    value: string;
    valueID?: string;
}
