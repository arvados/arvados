// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconTypes } from '../../icon/icon';
import AbstractItem from './abstract-item';
import EmptyState from '../../empty-state/empty-state';
import { EmptyResource } from '../../../models/empty';

export default class EmptyItem extends AbstractItem {

    constructor(item: EmptyResource) {
        super(item);
    }

    getIcon(): IconTypes {
        return IconTypes.FOLDER;
    }

    buildDetails(): React.ReactElement<any> {
        return <EmptyState icon={IconTypes.ANNOUNCEMENT}
            message='Select a file or folder to view its details.' />;
    }
}