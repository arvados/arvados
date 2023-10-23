// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DirectoryIcon, DefaultIcon, FileIcon } from "../icon/icon";

export const getIcon = (type: string) => {
    switch (type) {
        case 'directory':
            return DirectoryIcon;
        case 'file':
            return FileIcon;
        default:
            return DefaultIcon;
    }
};

