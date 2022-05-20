// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ProjectResource } from "./project";
import { CollectionResource } from "./collection";
import { ProcessResource } from "./process";
import { EmptyResource } from "./empty";
import { CollectionFile, CollectionDirectory } from 'models/collection-file';
import { WorkflowResource } from 'models/workflow';

export type DetailsResource = ProjectResource | CollectionResource | ProcessResource | EmptyResource | CollectionFile | CollectionDirectory | WorkflowResource;
