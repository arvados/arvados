// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from '../../models/kinds';
import ProjectItem from './items/project-item';
import CollectionItem from './items/collection-item';
import ProcessItem from './items/process-item';
import AbstractItem from './items/abstract-item';
import EmptyItem from './items/empty-item';
import { DetailsPanelResource } from '../../views-components/details-panel/details-panel';
import { EmptyResource } from '../../models/empty';

export default class DetailsPanelFactory {
    static createItem(res: DetailsPanelResource): AbstractItem {
        switch (res.kind) {
            case ResourceKind.Project:
                return new ProjectItem(res);
            case ResourceKind.Collection:
                return new CollectionItem(res);
            case ResourceKind.Process:
                return new ProcessItem(res);
            default:
                return new EmptyItem(res as EmptyResource);
        }
    }
}