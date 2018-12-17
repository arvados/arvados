// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type FileViewerList = FileViewer[];

export interface FileViewer {
    /**
     * Name is used as a label in file's context menu
     */
    name: string;

    /**
     * Limits files for which viewer is enabled
     * If not given, viewer will be enabled for all files
     * Viewer is enabled if file name ends with an extension.
     * 
     * Example: `['.zip', '.tar.gz', 'bam']`
     */
    extensions?: string[];

    /**
     * Determines whether a viewer is enabled for collections.
     */
    collections?: boolean;

    /**
     * URL that redirects to a viewer 
     * Example: `https://bam-viewer.com`
     */
    url: string;

    /**
     * Name of a search param that will be used to send file's path to a viewer
     * Example: 
     * 
     * `{ filePathParam: 'filePath' }`
     * 
     * `https://bam-viewer.com?filePath=/path/to/file`
     */
    filePathParam: string;

    /**
     * Icon that will display next to a label
     */
    iconUrl?: string;
}
