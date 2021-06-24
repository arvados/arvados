// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from 'axios';
import { FileViewerList } from 'models/file-viewers-config';

export class FileViewersConfigService {
    constructor(
        private url: string
    ) { }

    get() {
        return Axios
            .get<FileViewerList>(this.url)
            .then(response => response.data);
    }
}
