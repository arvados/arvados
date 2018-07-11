// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as classnames from "classnames";

import AccessTime from '@material-ui/icons/AccessTime';
import Announcement from '@material-ui/icons/Announcement';
import ArrowDropDown from '@material-ui/icons/ArrowDropDown';
import BubbleChart from '@material-ui/icons/BubbleChart';
import Cached from '@material-ui/icons/Cached';
import Code from '@material-ui/icons/Code';
import ChevronLeft from '@material-ui/icons/ChevronLeft';
import ChevronRight from '@material-ui/icons/ChevronRight';
import Close from '@material-ui/icons/Close';
import ContentCopy from '@material-ui/icons/ContentCopy';
import CreateNewFolder from '@material-ui/icons/CreateNewFolder';
import Delete from '@material-ui/icons/Delete';
import Edit from '@material-ui/icons/Edit';
import FolderIcon from '@material-ui/icons/Folder';
import GetApp from '@material-ui/icons/GetApp';
import Help from '@material-ui/icons/Help';
import Inbox from '@material-ui/icons/Inbox';
import Info from '@material-ui/icons/Info';
import Input from '@material-ui/icons/Input';
import Menu from '@material-ui/icons/Menu';
import MoreVert from '@material-ui/icons/MoreVert';
import NotificationsIcon from '@material-ui/icons/Notifications';
import People from '@material-ui/icons/People';
import Person from '@material-ui/icons/Person';
import PersonAdd from '@material-ui/icons/PersonAdd';
import PlayArrow from '@material-ui/icons/PlayArrow';
import RateReview from '@material-ui/icons/RateReview';
import Search from '@material-ui/icons/Search';
import Star from '@material-ui/icons/Star';
import StarBorder from '@material-ui/icons/StarBorder';

export enum IconTypes {
    ACCESS_TIME = 'access_time',
    ANNOUNCEMENT = 'announcement',
    ARROW_DROP_DOWN = 'arrow_drop_down',
    BUBBLE_CHART = 'bubble_chart',
    CACHED = 'cached',
    CODE = 'code',
    CHEVRON_LEFT = 'chevron_left',
    CHEVRON_RIGHT = 'chevron-right',
    COLLECTION = 'collection',
    CLOSE = 'close',
    CONTENT_COPY = 'content_copy',
    CREATE_NEW_FOLDER = 'create_new_folder',
    DELETE = 'delete',
    EDIT = 'edit',
    FOLDER = 'folder',
    GET_APP = 'get_app',
    HELP = 'help',
    INBOX = 'inbox',
    INFO = 'info',
    INPUT = 'input',
    MENU = 'menu',
    MORE_VERT = 'more_vert',
    NOTIFICATIONS = 'notifications',
    PEOPLE = 'people',
    PERSON = 'person',
    PERSON_ADD = 'person_add',
    PLAY_ARROW = 'play_arrow',
    RATE_REVIEW = 'rate_review',
    SEARCH = 'search',
    STAR = 'star',
    STAR_BORDER = 'star_border'
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
    [IconTypes.ANNOUNCEMENT]: <Announcement className={props.className} />,
    [IconTypes.ARROW_DROP_DOWN]: <ArrowDropDown className={props.className} />,
    [IconTypes.BUBBLE_CHART]: <BubbleChart className={props.className} />,
    [IconTypes.CACHED]: <Cached className={props.className} />,
    [IconTypes.CODE]: <Code className={props.className} />,
    [IconTypes.CHEVRON_LEFT]: <ChevronLeft className={props.className} />,
    [IconTypes.CHEVRON_RIGHT]: <ChevronRight className={props.className} />,
    [IconTypes.COLLECTION]: <i className={classnames([props.className, 'fas fa-archive fa-lg'])} />,
    [IconTypes.CLOSE]: <Close className={props.className} />,
    [IconTypes.CONTENT_COPY]: <ContentCopy className={props.className} />,
    [IconTypes.CREATE_NEW_FOLDER]: <CreateNewFolder className={props.className} />,
    [IconTypes.DELETE]: <Delete className={props.className} />,
    [IconTypes.EDIT]: <Edit className={props.className} />,    
    [IconTypes.FOLDER]: <FolderIcon className={props.className} />,
    [IconTypes.GET_APP]: <GetApp className={props.className} />,
    [IconTypes.HELP]: <Help className={props.className} />,
    [IconTypes.INBOX]: <Inbox className={props.className} />,
    [IconTypes.INFO]: <Info className={props.className} />,
    [IconTypes.INPUT]: <Input className={props.className} />,
    [IconTypes.MENU]: <Menu className={props.className} />,
    [IconTypes.MORE_VERT]: <MoreVert className={props.className} />,
    [IconTypes.NOTIFICATIONS]: <NotificationsIcon className={props.className} />,
    [IconTypes.PEOPLE]: <People className={props.className} />,
    [IconTypes.PERSON]: <Person className={props.className} />,
    [IconTypes.PERSON_ADD]: <PersonAdd className={props.className} />,
    [IconTypes.PLAY_ARROW]: <PlayArrow className={props.className} />,
    [IconTypes.RATE_REVIEW]: <RateReview className={props.className} />,
    [IconTypes.SEARCH]: <Search className={props.className} />,
    [IconTypes.STAR]: <Star className={props.className} />,
    [IconTypes.STAR_BORDER]: <StarBorder className={props.className} />
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