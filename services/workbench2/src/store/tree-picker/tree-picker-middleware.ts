// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource } from "models/link";
import { UserResource } from "models/user";
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { CollectionDirectory, CollectionFile } from 'models/collection-file';

export interface ProjectsTreePickerRootItem {
    id: string;
    name: string;
}

export type ProjectsTreePickerItem = ProjectsTreePickerRootItem | GroupContentsResource | CollectionDirectory | CollectionFile | LinkResource | UserResource;
