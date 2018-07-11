

import * as React from 'react';
import { ProjectResource } from "../../models/project";
import { CollectionResource } from "../../models/collection";
import { ProcessResource } from "../../models/process";
import { ResourceKind } from '../../models/kinds';
import ProjectItem from './items/project-item';
import CollectionItem from './items/collection-item';
import ProcessItem from './items/process-item';
import { AbstractItem } from './items/abstract-item';

// TODO: move to models
export type DetailsPanelResource = ProjectResource | CollectionResource | ProcessResource;

export default class DetailsPanelFactory {
    static createItem(res: DetailsPanelResource): AbstractItem {
        switch (res.kind) {
            case ResourceKind.Project:
                return new ProjectItem(res);
            case ResourceKind.Collection:
                return new CollectionItem(res);
            case ResourceKind.Collection:
                return new ProcessItem(res);
            default:
                return new ProjectItem(res);
        }
    }
}