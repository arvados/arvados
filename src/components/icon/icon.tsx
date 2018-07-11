// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as classnames from "classnames";

import AccessTime from '@material-ui/icons/AccessTime';
import CodeIcon from '@material-ui/icons/Code';
import CloseAnnouncement from '@material-ui/icons/Announcement';
import CloseIcon from '@material-ui/icons/Close';
import DeleteIcon from '@material-ui/icons/Delete';
import FolderIcon from '@material-ui/icons/Folder';
import InboxIcon from '@material-ui/icons/Inbox';
import InfoIcon from '@material-ui/icons/Info';
import HelpIcon from '@material-ui/icons/Help';
import NotificationsIcon from '@material-ui/icons/Notifications';
import PeopleIcon from '@material-ui/icons/People';
import PersonIcon from '@material-ui/icons/Person';
import PlayArrow from '@material-ui/icons/PlayArrow';
import StarIcon from '@material-ui/icons/Star';

export enum IconTypes {
    ACCESS_TIME = 'access_time',
    ANNOUNCEMENT = 'announcement',
    CODE = 'code',
    COLLECTION = 'collection',
    CLOSE = 'close',
    DELETE = 'delete',
    FOLDER = 'folder',
    HELP = 'help',
    INBOX = 'inbox',
    INFO = 'info',
    NOTIFICATIONS = 'notifications',
    PEOPLE = 'people',
    PERSON = 'person',
    PLAY_ARROW = 'play_arrow',
    PROCESS = 'process',
    PROJECT  = 'project',
    STAR = 'star'
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
    [IconTypes.ACCESS_TIME]: <AccessTime className={props.className} />,
    [IconTypes.ANNOUNCEMENT]: <CloseAnnouncement className={props.className} />,
    [IconTypes.CODE]: <CodeIcon className={props.className} />,
    [IconTypes.COLLECTION]: <i className={classnames([props.className, 'fas fa-archive fa-lg'])} />,
    [IconTypes.CLOSE]: <CloseIcon className={props.className} />,
    [IconTypes.DELETE]: <DeleteIcon className={props.className} />,
    [IconTypes.FOLDER]: <FolderIcon className={props.className} />,
    [IconTypes.HELP]: <HelpIcon className={props.className} />,
    [IconTypes.INFO]: <InfoIcon className={props.className} />,
    [IconTypes.INBOX]: <InboxIcon className={props.className} />,
    [IconTypes.NOTIFICATIONS]: <NotificationsIcon className={props.className} />,
    [IconTypes.PEOPLE]: <PeopleIcon className={props.className} />,
    [IconTypes.PERSON]: <PersonIcon className={props.className} />,
    [IconTypes.PLAY_ARROW]: <PlayArrow className={props.className} />,
    [IconTypes.PROCESS]: <i className={classnames([props.className, 'fas fa-cogs fa-lg'])} />,
    [IconTypes.PROJECT]: <i className={classnames([props.className, 'fas fa-folder fa-lg'])} />,
    [IconTypes.STAR]: <StarIcon className={props.className} />
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