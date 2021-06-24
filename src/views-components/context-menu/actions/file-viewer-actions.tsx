// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ListItemText, ListItem, ListItemIcon, Icon } from "@material-ui/core";
import { RootState } from 'store/store';
import { getNodeValue } from 'models/tree';
import { CollectionDirectory, CollectionFile, CollectionFileType } from 'models/collection-file';
import { FileViewerList, FileViewer } from 'models/file-viewers-config';
import { getFileViewers } from 'store/file-viewers/file-viewers-selectors';
import { connect } from 'react-redux';
import { OpenIcon } from 'components/icon/icon';

interface FileViewerActionProps {
    fileUrl: string;
    viewers: FileViewerList;
}

const mapStateToProps = (state: RootState): FileViewerActionProps => {
    const { resource } = state.contextMenu;
    if (resource) {
        const file = getNodeValue(resource.uuid)(state.collectionPanelFiles);
        if (file) {
            const fileViewers = getFileViewers(state.properties);
            return {
                fileUrl: file.url,
                viewers: fileViewers.filter(enabledViewers(file)),
            };
        }
    }
    return {
        fileUrl: '',
        viewers: [],
    };
};

const enabledViewers = (file: CollectionFile | CollectionDirectory) =>
    ({ extensions, collections }: FileViewer) => {
        if (collections && file.type === CollectionFileType.DIRECTORY) {
            return true;
        } else if (extensions) {
            return extensions.some(extension => file.name.endsWith(extension));
        } else {
            return true;
        }
    };

const fillViewerUrl = (fileUrl: string, { url, filePathParam }: FileViewer) => {
    const viewerUrl = new URL(url);
    viewerUrl.searchParams.append(filePathParam, fileUrl);
    return viewerUrl.href;
};

export const FileViewerActions = connect(mapStateToProps)(
    ({ fileUrl, viewers, onClick }: FileViewerActionProps & { onClick: () => void }) =>
        <>
            {viewers.map(viewer =>
                <ListItem
                    button
                    component='a'
                    key={viewer.name}
                    style={{ textDecoration: 'none' }}
                    href={fillViewerUrl(fileUrl, viewer)}
                    onClick={onClick}
                    target='_blank'>
                    <ListItemIcon>
                        {
                            viewer.iconUrl
                                ? <Icon>
                                    <img src={viewer.iconUrl} />
                                </Icon>
                                : <OpenIcon />
                        }
                    </ListItemIcon>
                    <ListItemText>
                        {viewer.name}
                    </ListItemText>
                </ListItem>
            )}
        </>);
