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
    access_time: <AccessTime className={props.className} />,
    announcement: <CloseAnnouncement className={props.className} />,
    code: <CodeIcon className={props.className} />,
    collection: <i className={classnames([props.className, 'fas fa-archive fa-lg'])} />,
    close: <CloseIcon className={props.className} />,
    delete: <DeleteIcon className={props.className} />,
    folder: <FolderIcon className={props.className} />,
    help: <HelpIcon className={props.className} />,
    info: <InfoIcon className={props.className} />,
    inbox: <InboxIcon className={props.className} />,
    notifications: <NotificationsIcon className={props.className} />,
    people: <PeopleIcon className={props.className} />,
    person: <PersonIcon className={props.className} />,
    play_arrow: <PlayArrow className={props.className} />,
    process: <i className={classnames([props.className, 'fas fa-cogs fa-lg'])} />,
    project: <i className={classnames([props.className, 'fas fa-folder fa-lg'])} />,
    star: <StarIcon className={props.className} />
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