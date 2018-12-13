// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PropertiesState, getProperty } from '~/store/properties/properties';
import { FileViewerList } from '~/models/file-viewers-config';

export const FILE_VIEWERS_PROPERTY_NAME = 'fileViewers';

export const DEFAULT_FILE_VIEWERS: FileViewerList = [];
export const getFileViewers = (state: PropertiesState) =>
    getProperty<FileViewerList>(FILE_VIEWERS_PROPERTY_NAME)(state) || DEFAULT_FILE_VIEWERS;
