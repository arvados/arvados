// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Badge, SvgIcon, Tooltip } from '@material-ui/core';
import Add from '@material-ui/icons/Add';
import ArrowBack from '@material-ui/icons/ArrowBack';
import ArrowDropDown from '@material-ui/icons/ArrowDropDown';
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
import MoveToInbox from '@material-ui/icons/MoveToInbox';
import Info from '@material-ui/icons/Info';
import Input from '@material-ui/icons/Input';
import InsertDriveFile from '@material-ui/icons/InsertDriveFile';
import LastPage from '@material-ui/icons/LastPage';
import LibraryBooks from '@material-ui/icons/LibraryBooks';
import ListAlt from '@material-ui/icons/ListAlt';
import Menu from '@material-ui/icons/Menu';
import MoreVert from '@material-ui/icons/MoreVert';
import Mail from '@material-ui/icons/Mail';
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
import Settings from '@material-ui/icons/Settings';
import Star from '@material-ui/icons/Star';
import StarBorder from '@material-ui/icons/StarBorder';
import Warning from '@material-ui/icons/Warning';
import VpnKey from '@material-ui/icons/VpnKey';
import LinkOutlined from '@material-ui/icons/LinkOutlined';
import RemoveRedEye from '@material-ui/icons/RemoveRedEye';
import Computer from '@material-ui/icons/Computer';
import WrapText from '@material-ui/icons/WrapText';
import TextIncrease from '@material-ui/icons/ZoomIn';
import TextDecrease from '@material-ui/icons/ZoomOut';
import FullscreenSharp from '@material-ui/icons/FullscreenSharp';
import FullscreenExitSharp from '@material-ui/icons/FullscreenExitSharp';
import ExitToApp from '@material-ui/icons/ExitToApp';
import CheckCircleOutline from '@material-ui/icons/CheckCircleOutline';
import RemoveCircleOutline from '@material-ui/icons/RemoveCircleOutline';
import NotInterested from '@material-ui/icons/NotInterested';
import Image from '@material-ui/icons/Image';

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

export const FreezeIcon = (props: any) =>
    <SvgIcon {...props}>
        <path d="M20.79,13.95L18.46,14.57L16.46,13.44V10.56L18.46,9.43L20.79,10.05L21.31,8.12L19.54,7.65L20,5.88L18.07,5.36L17.45,7.69L15.45,8.82L13,7.38V5.12L14.71,3.41L13.29,2L12,3.29L10.71,2L9.29,3.41L11,5.12V7.38L8.5,8.82L6.5,7.69L5.92,5.36L4,5.88L4.47,7.65L2.7,8.12L3.22,10.05L5.55,9.43L7.55,10.56V13.45L5.55,14.58L3.22,13.96L2.7,15.89L4.47,16.36L4,18.12L5.93,18.64L6.55,16.31L8.55,15.18L11,16.62V18.88L9.29,20.59L10.71,22L12,20.71L13.29,22L14.7,20.59L13,18.88V16.62L15.5,15.17L17.5,16.3L18.12,18.63L20,18.12L19.53,16.35L21.3,15.88L20.79,13.95M9.5,10.56L12,9.11L14.5,10.56V13.44L12,14.89L9.5,13.44V10.56Z" />
    </SvgIcon>

export const UnfreezeIcon = (props: any) =>
    <SvgIcon {...props}>
        <path d="M11 5.12L9.29 3.41L10.71 2L12 3.29L13.29 2L14.71 3.41L13 5.12V7.38L15.45 8.82L17.45 7.69L18.07 5.36L20 5.88L19.54 7.65L21.31 8.12L20.79 10.05L18.46 9.43L16.46 10.56V13.26L14.5 11.3V10.56L12.74 9.54L10.73 7.53L11 7.38V5.12M18.46 14.57L16.87 13.67L19.55 16.35L21.3 15.88L20.79 13.95L18.46 14.57M13 16.62V18.88L14.7 20.59L13.29 22L12 20.71L10.71 22L9.29 20.59L11 18.88V16.62L8.55 15.18L6.55 16.31L5.93 18.64L4 18.12L4.47 16.36L2.7 15.89L3.22 13.96L5.55 14.58L7.55 13.45V10.56L5.55 9.43L3.22 10.05L2.7 8.12L4.47 7.65L4 5.89L1.11 3L2.39 1.73L22.11 21.46L20.84 22.73L14.1 16L13 16.62M12 14.89L12.63 14.5L9.5 11.39V13.44L12 14.89Z" />
    </SvgIcon>

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

// https://materialdesignicons.com/icon/image-off
export const ImageOffIcon = (props: any) =>
    <SvgIcon {...props}>
        <path d="M21 17.2L6.8 3H19C20.1 3 21 3.9 21 5V17.2M20.7 22L19.7 21H5C3.9 21 3 20.1 3 19V4.3L2 3.3L3.3 2L22 20.7L20.7 22M16.8 18L12.9 14.1L11 16.5L8.5 13.5L5 18H16.8Z" />
    </SvgIcon>;

// https://materialdesignicons.com/icon/inbox-arrow-up
export const OutputIcon: IconType = (props: any) =>
    <SvgIcon {...props}>
        <path d="M14,14H10V11H8L12,7L16,11H14V14M16,11M5,15V5H19V15H15A3,3 0 0,1 12,18A3,3 0 0,1 9,15H5M19,3H5C3.89,3 3,3.9 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3" />
    </SvgIcon>;

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
export const FileInputIcon: IconType = (props) => <InsertDriveFile {...props} />;
export const KeyIcon: IconType = (props) => <VpnKey {...props} />;
export const LogIcon: IconType = (props) => <SettingsEthernet {...props} />;
export const MailIcon: IconType = (props) => <Mail {...props} />;
export const MaximizeIcon: IconType = (props) => <FullscreenSharp {...props} />;
export const UnMaximizeIcon: IconType = (props) => <FullscreenExitSharp {...props} />;
export const MoreOptionsIcon: IconType = (props) => <MoreVert {...props} />;
export const MoveToIcon: IconType = (props) => <Input {...props} />;
export const NewProjectIcon: IconType = (props) => <CreateNewFolder {...props} />;
export const NotificationIcon: IconType = (props) => <Notifications {...props} />;
export const OpenIcon: IconType = (props) => <OpenInNew {...props} />;
export const InputIcon: IconType = (props) => <MoveToInbox {...props} />;
export const PaginationDownIcon: IconType = (props) => <ArrowDropDown {...props} />;
export const PaginationLeftArrowIcon: IconType = (props) => <ChevronLeft {...props} />;
export const PaginationRightArrowIcon: IconType = (props) => <ChevronRight {...props} />;
export const ProcessIcon: IconType = (props) => <Settings {...props} />;
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
export const ImageIcon: IconType = (props) => <Image {...props} />;
