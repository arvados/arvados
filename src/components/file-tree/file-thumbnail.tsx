// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import isImage from 'is-image';
import { withStyles, WithStyles } from '@material-ui/core';
import { FileTreeData } from '~/components/file-tree/file-tree-data';
import { CollectionFileType } from '~/models/collection-file';

export interface FileThumbnailProps {
    file: FileTreeData;
}

export const FileThumbnail =
    ({ file }: FileThumbnailProps) =>
        file.type === CollectionFileType.FILE && isImage(file.name)
            ? <ImageFileThumbnail file={file} />
            : null;

type ImageFileThumbnailCssRules = 'thumbnail';

const imageFileThumbnailStyle = withStyles<ImageFileThumbnailCssRules>(theme => ({
    thumbnail: {
        maxWidth: 250,
        margin: `${theme.spacing.unit}px 0`,
    }
}));

const ImageFileThumbnail = imageFileThumbnailStyle(
    ({ classes, file }: WithStyles<ImageFileThumbnailCssRules> & FileThumbnailProps) =>
        <img
            className={classes.thumbnail}
            alt={file.name}
            src={file.url} />
);
