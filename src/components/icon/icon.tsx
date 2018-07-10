// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as classnames from "classnames";
import CloseAnnouncement from '@material-ui/icons/Announcement';
import CloseIcon from '@material-ui/icons/Close';
import FolderIcon from '@material-ui/icons/Folder';

export enum IconTypes {
    ANNOUNCEMENT = 'announcement',
    FOLDER = 'folder',
    CLOSE = 'close',
    PROJECT  = 'project',
    COLLECTION = 'collection',
    PROCESS = 'process'
}

interface IconBaseDataProps {
    icon: IconTypes;
    className?: string;
}

type IconBaseProps = IconBaseDataProps;

interface IconBaseState {
    icon: IconTypes;
}

const getSpecificIcon = (props: any) => ({
    announcement: <CloseAnnouncement className={props.className} />,
    folder: <FolderIcon className={props.className} />,
    close: <CloseIcon className={props.className} />,
    project: <i className={classnames([props.className, 'fas fa-folder fa-lg'])} />,
    collection: <i className={classnames([props.className, 'fas fa-archive fa-lg'])} />,
    process: <i className={classnames([props.className, 'fas fa-cogs fa-lg'])} />
});

class IconBase extends React.Component<IconBaseProps, IconBaseState> {
    state = {
        icon: IconTypes.FOLDER,
    };

    render() {
        return getSpecificIcon(this.props)[this.props.icon];
    }
}

export default IconBase;