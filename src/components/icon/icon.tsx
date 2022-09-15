// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Badge, Tooltip } from '@material-ui/core';
import Add from '@material-ui/icons/Add';
import ArrowBack from '@material-ui/icons/ArrowBack';
import ArrowDropDown from '@material-ui/icons/ArrowDropDown';
import BubbleChart from '@material-ui/icons/BubbleChart';
import Build from '@material-ui/icons/Build';
import Cached from '@material-ui/icons/Cached';
import DescriptionIcon from '@material-ui/icons/Description';
import ChevronLeft from '@material-ui/icons/ChevronLeft';
import CloudUpload from '@material-ui/icons/CloudUpload';
import Code from '@material-ui/icons/Code';
import Create from '@material-ui/icons/Create';
import ImportContacts from '@material-ui/icons/ImportContacts';
import ChevronRight from '@material-ui/icons/ChevronRight';
import Close from '@material-ui/icons/Close';
import ContentCopy from '@material-ui/icons/FileCopyOutlined';
import CreateNewFolder from '@material-ui/icons/CreateNewFolder';
import Delete from '@material-ui/icons/Delete';
import DeviceHub from '@material-ui/icons/DeviceHub';
import Edit from '@material-ui/icons/Edit';
import ErrorRoundedIcon from '@material-ui/icons/ErrorRounded';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import FlipToFront from '@material-ui/icons/FlipToFront';
import Folder from '@material-ui/icons/Folder';
import FolderShared from '@material-ui/icons/FolderShared';
import Pageview from '@material-ui/icons/Pageview';
import GetApp from '@material-ui/icons/GetApp';
import Help from '@material-ui/icons/Help';
import HelpOutline from '@material-ui/icons/HelpOutline';
import History from '@material-ui/icons/History';
import Inbox from '@material-ui/icons/Inbox';
import Info from '@material-ui/icons/Info';
import Input from '@material-ui/icons/Input';
import InsertDriveFile from '@material-ui/icons/InsertDriveFile';
import LastPage from '@material-ui/icons/LastPage';
import LibraryBooks from '@material-ui/icons/LibraryBooks';
import ListAlt from '@material-ui/icons/ListAlt';
import Menu from '@material-ui/icons/Menu';
import MoreVert from '@material-ui/icons/MoreVert';
import Mail from '@material-ui/icons/Mail';
import MoveToInbox from '@material-ui/icons/MoveToInbox';
import Notifications from '@material-ui/icons/Notifications';
import OpenInNew from '@material-ui/icons/OpenInNew';
import People from '@material-ui/icons/People';
import Person from '@material-ui/icons/Person';
import PersonAdd from '@material-ui/icons/PersonAdd';
import PlayArrow from '@material-ui/icons/PlayArrow';
import Public from '@material-ui/icons/Public';
import RateReview from '@material-ui/icons/RateReview';
import RestoreFromTrash from '@material-ui/icons/History';
import Search from '@material-ui/icons/Search';
import SettingsApplications from '@material-ui/icons/SettingsApplications';
import SettingsEthernet from '@material-ui/icons/SettingsEthernet';
import Star from '@material-ui/icons/Star';
import StarBorder from '@material-ui/icons/StarBorder';
import Warning from '@material-ui/icons/Warning';
import Visibility from '@material-ui/icons/Visibility';
import VisibilityOff from '@material-ui/icons/VisibilityOff';
import VpnKey from '@material-ui/icons/VpnKey';
import LinkOutlined from '@material-ui/icons/LinkOutlined';
import RemoveRedEye from '@material-ui/icons/RemoveRedEye';
import Computer from '@material-ui/icons/Computer';
import WrapText from '@material-ui/icons/WrapText';
import TextIncrease from '@material-ui/icons/ZoomIn';
import TextDecrease from '@material-ui/icons/ZoomOut';
import CropFreeSharp from '@material-ui/icons/CropFreeSharp';
import ExitToApp from '@material-ui/icons/ExitToApp';
import CheckCircleOutline from '@material-ui/icons/CheckCircleOutline';
import RemoveCircleOutline from '@material-ui/icons/RemoveCircleOutline';
import NotInterested from '@material-ui/icons/NotInterested';

// Import FontAwesome icons
import { library } from '@fortawesome/fontawesome-svg-core';
import { faPencilAlt, faSlash, faUsers, faEllipsisH } from '@fortawesome/free-solid-svg-icons';
import { FormatAlignLeft } from '@material-ui/icons';
library.add(
    faPencilAlt,
    faSlash,
    faUsers,
    faEllipsisH,
);

export const PendingIcon = (props: any) =>
    <span {...props}>
        <span className='fas fa-ellipsis-h' />
    </span>

export const ReadOnlyIcon = (props: any) =>
    <span {...props}>
        <div className="fa-layers fa-1x fa-fw">
            <span className="fas fa-slash"
                data-fa-mask="fas fa-pencil-alt" data-fa-transform="down-1.5" />
            <span className="fas fa-slash" />
        </div>
    </span>;

export const GroupsIcon = (props: any) =>
    <span {...props}>
        <span className="fas fa-users" />
    </span>;

export const CollectionOldVersionIcon = (props: any) =>
    <Tooltip title='Old version'>
        <Badge badgeContent={<History fontSize='small' />}>
            <CollectionIcon {...props} />
        </Badge>
    </Tooltip>;

export type IconType = React.SFC<{ className?: string, style?: object }>;

export const AddIcon: IconType = (props) => <Add {...props} />;
export const AddFavoriteIcon: IconType = (props) => <StarBorder {...props} />;
export const AdminMenuIcon: IconType = (props) => <Build {...props} />;
export const AdvancedIcon: IconType = (props) => <SettingsApplications {...props} />;
export const AttributesIcon: IconType = (props) => <ListAlt {...props} />;
export const BackIcon: IconType = (props) => <ArrowBack {...props} />;
export const CustomizeTableIcon: IconType = (props) => <Menu {...props} />;
export const CommandIcon: IconType = (props) => <LastPage {...props} />;
export const CopyIcon: IconType = (props) => <ContentCopy {...props} />;
export const CollectionIcon: IconType = (props) => <LibraryBooks {...props} />;
export const CloseIcon: IconType = (props) => <Close {...props} />;
export const CloudUploadIcon: IconType = (props) => <CloudUpload {...props} />;
export const DefaultIcon: IconType = (props) => <RateReview {...props} />;
export const DetailsIcon: IconType = (props) => <Info {...props} />;
export const DirectoryIcon: IconType = (props) => <Folder {...props} />;
export const DownloadIcon: IconType = (props) => <GetApp {...props} />;
export const EditSavedQueryIcon: IconType = (props) => <Create {...props} />;
export const ExpandIcon: IconType = (props) => <ExpandMoreIcon {...props} />;
export const ErrorIcon: IconType = (props) => <ErrorRoundedIcon style={{ color: '#ff0000' }} {...props} />;
export const FavoriteIcon: IconType = (props) => <Star {...props} />;
export const FileIcon: IconType = (props) => <DescriptionIcon {...props} />;
export const HelpIcon: IconType = (props) => <Help {...props} />;
export const HelpOutlineIcon: IconType = (props) => <HelpOutline {...props} />;
export const ImportContactsIcon: IconType = (props) => <ImportContacts {...props} />;
export const InfoIcon: IconType = (props) => <Info {...props} />;
export const InputIcon: IconType = (props) => <InsertDriveFile {...props} />;
export const KeyIcon: IconType = (props) => <VpnKey {...props} />;
export const LogIcon: IconType = (props) => <SettingsEthernet {...props} />;
export const MailIcon: IconType = (props) => <Mail {...props} />;
export const MaximizeIcon: IconType = (props) => <CropFreeSharp {...props} />;
export const MoreOptionsIcon: IconType = (props) => <MoreVert {...props} />;
export const MoveToIcon: IconType = (props) => <Input {...props} />;
export const NewProjectIcon: IconType = (props) => <CreateNewFolder {...props} />;
export const NotificationIcon: IconType = (props) => <Notifications {...props} />;
export const OpenIcon: IconType = (props) => <OpenInNew {...props} />;
export const OutputIcon: IconType = (props) => <MoveToInbox {...props} />;
export const PaginationDownIcon: IconType = (props) => <ArrowDropDown {...props} />;
export const PaginationLeftArrowIcon: IconType = (props) => <ChevronLeft {...props} />;
export const PaginationRightArrowIcon: IconType = (props) => <ChevronRight {...props} />;
export const ProcessIcon: IconType = (props) => <BubbleChart {...props} />;
export const ProjectIcon: IconType = (props) => <Folder {...props} />;
export const FilterGroupIcon: IconType = (props) => <Pageview {...props} />;
export const ProjectsIcon: IconType = (props) => <Inbox {...props} />;
export const ProvenanceGraphIcon: IconType = (props) => <DeviceHub {...props} />;
export const RemoveIcon: IconType = (props) => <Delete {...props} />;
export const RemoveFavoriteIcon: IconType = (props) => <Star {...props} />;
export const PublicFavoriteIcon: IconType = (props) => <Public {...props} />;
export const RenameIcon: IconType = (props) => <Edit {...props} />;
export const RestoreVersionIcon: IconType = (props) => <FlipToFront {...props} />;
export const RestoreFromTrashIcon: IconType = (props) => <RestoreFromTrash {...props} />;
export const ReRunProcessIcon: IconType = (props) => <Cached {...props} />;
export const SearchIcon: IconType = (props) => <Search {...props} />;
export const ShareIcon: IconType = (props) => <PersonAdd {...props} />;
export const ShareMeIcon: IconType = (props) => <People {...props} />;
export const SidePanelRightArrowIcon: IconType = (props) => <PlayArrow {...props} />;
export const TrashIcon: IconType = (props) => <Delete {...props} />;
export const UserPanelIcon: IconType = (props) => <Person {...props} />;
export const UsedByIcon: IconType = (props) => <Folder {...props} />;
export const WorkflowIcon: IconType = (props) => <Code {...props} />;
export const WarningIcon: IconType = (props) => <Warning style={{ color: '#fbc02d', height: '30px', width: '30px' }} {...props} />;
export const Link: IconType = (props) => <LinkOutlined {...props} />;
export const FolderSharedIcon: IconType = (props) => <FolderShared {...props} />;
export const CanReadIcon: IconType = (props) => <RemoveRedEye {...props} />;
export const CanWriteIcon: IconType = (props) => <Edit {...props} />;
export const CanManageIcon: IconType = (props) => <Computer {...props} />;
export const AddUserIcon: IconType = (props) => <PersonAdd {...props} />;
export const WordWrapOnIcon: IconType = (props) => <WrapText {...props} />;
export const WordWrapOffIcon: IconType = (props) => <FormatAlignLeft {...props} />;
export const TextIncreaseIcon: IconType = (props) => <TextIncrease {...props} />;
export const TextDecreaseIcon: IconType = (props) => <TextDecrease {...props} />;
export const DeactivateUserIcon: IconType = (props) => <NotInterested {...props} />;
export const LoginAsIcon: IconType = (props) => <ExitToApp {...props} />;
export const ActiveIcon: IconType = (props) => <CheckCircleOutline {...props} />;
export const SetupIcon: IconType = (props) => <RemoveCircleOutline {...props} />;
export const InactiveIcon: IconType = (props) => <NotInterested {...props} />;
