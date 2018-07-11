// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconTypes } from '../../icon/icon';
import AbstractItem from './abstract-item';
import EmptyState from '../../empty-state/empty-state';
import { EmptyResource } from '../../../models/empty';

export default class EmptyItem extends AbstractItem<EmptyResource> {
    
    getIcon(): IconTypes {
        return IconTypes.INBOX;
    }

    buildDetails(): React.ReactElement<any> {
        return <EmptyState icon={IconTypes.RATE_REVIEW}
            message='Select a file or folder to view its details.' />;
    }
}