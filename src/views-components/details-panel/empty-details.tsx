// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DefaultIcon, ProjectsIcon } from 'components/icon/icon';
import { EmptyResource } from 'models/empty';
import { DetailsData } from "./details-data";
import { DefaultView } from 'components/default-view/default-view';

export class EmptyDetails extends DetailsData<EmptyResource> {
    getIcon(className?: string) {
        return <ProjectsIcon className={className}/>;
    }

    getDetails() {
        return <DefaultView icon={DefaultIcon} messages={['Select a file or folder to view its details.']} />;
    }
}
