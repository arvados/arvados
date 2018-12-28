// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectIcon } from '~/components/icon/icon';
import { DetailsData } from "./details-data";
import { CollectionFile, CollectionDirectory } from '~/models/collection-file';

export class FileDetails extends DetailsData<CollectionFile | CollectionDirectory> {

    getIcon(className?: string){
        return <ProjectIcon className={className} />;
    }

    getDetails() {
        return <div>File details</div>;
    }
}
