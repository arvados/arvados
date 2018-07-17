// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DefaultIcon, ProjectsIcon } from '../../icon/icon';
import AbstractItem from './abstract-item';
import EmptyState from '../../empty-state/empty-state';
import { EmptyResource } from '../../../models/empty';

export default class EmptyItem extends AbstractItem<EmptyResource> {
    
    getIcon(className?: string) {
        return <ProjectsIcon className={className} />;
    }

    buildDetails() {
        return <EmptyState icon={DefaultIcon}
            message='Select a file or folder to view its details.' />;
    }
}