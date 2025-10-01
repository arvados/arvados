// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Badge, SvgIcon, Tooltip } from "@mui/material";
import Add from "@mui/icons-material/Add";
import ArrowBack from "@mui/icons-material/ArrowBack";
import ArrowDropDown from "@mui/icons-material/ArrowDropDown";
import Build from "@mui/icons-material/Build";
import Cached from "@mui/icons-material/Cached";
import DescriptionIcon from "@mui/icons-material/Description";
import ChevronLeft from "@mui/icons-material/ChevronLeft";
import CloudUpload from "@mui/icons-material/CloudUpload";
import Code from "@mui/icons-material/Code";
import Create from "@mui/icons-material/Create";
import ImportContacts from "@mui/icons-material/ImportContacts";
import ChevronRight from "@mui/icons-material/ChevronRight";
import Close from "@mui/icons-material/Close";
import ContentCopy from "@mui/icons-material/ContentCopy";
import FileCopyOutlined from "@mui/icons-material/FileCopyOutlined";
import CreateNewFolder from "@mui/icons-material/CreateNewFolder";
import Delete from "@mui/icons-material/Delete";
import DeviceHub from "@mui/icons-material/DeviceHub";
import Edit from "@mui/icons-material/Edit";
import ErrorRoundedIcon from "@mui/icons-material/ErrorRounded";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import FlipToFront from "@mui/icons-material/FlipToFront";
import Folder from "@mui/icons-material/Folder";
import FolderShared from "@mui/icons-material/FolderShared";
import Pageview from "@mui/icons-material/Pageview";
import GetApp from "@mui/icons-material/GetApp";
import Help from "@mui/icons-material/Help";
import HelpOutline from "@mui/icons-material/HelpOutline";
import History from "@mui/icons-material/History";
import Inbox from "@mui/icons-material/Inbox";
import Memory from "@mui/icons-material/Memory";
import MoveToInbox from "@mui/icons-material/MoveToInbox";
import Info from "@mui/icons-material/Info";
import Input from "@mui/icons-material/Input";
import InsertDriveFile from "@mui/icons-material/InsertDriveFile";
import LastPage from "@mui/icons-material/LastPage";
import LibraryBooks from "@mui/icons-material/LibraryBooks";
import ListAlt from "@mui/icons-material/ListAlt";
import Menu from "@mui/icons-material/Menu";
import MoreVert from "@mui/icons-material/MoreVert";
import MoreHoriz from "@mui/icons-material/MoreHoriz";
import Mail from "@mui/icons-material/Mail";
import Notifications from "@mui/icons-material/Notifications";
import OpenInNew from "@mui/icons-material/OpenInNew";
import People from "@mui/icons-material/People";
import Person from "@mui/icons-material/Person";
import PersonAdd from "@mui/icons-material/PersonAdd";
import PlayArrow from "@mui/icons-material/PlayArrow";
import Public from "@mui/icons-material/Public";
import RateReview from "@mui/icons-material/RateReview";
import RestoreFromTrash from "@mui/icons-material/History";
import Search from "@mui/icons-material/Search";
import SettingsApplications from "@mui/icons-material/SettingsApplications";
import SettingsEthernet from "@mui/icons-material/SettingsEthernet";
import Settings from "@mui/icons-material/Settings";
import Star from "@mui/icons-material/Star";
import StarBorder from "@mui/icons-material/StarBorder";
import Warning from "@mui/icons-material/Warning";
import VpnKey from "@mui/icons-material/VpnKey";
import LinkOutlined from "@mui/icons-material/LinkOutlined";
import RemoveRedEye from "@mui/icons-material/RemoveRedEye";
import Computer from "@mui/icons-material/Computer";
import WrapText from "@mui/icons-material/WrapText";
import TextIncrease from "@mui/icons-material/ZoomIn";
import TextDecrease from "@mui/icons-material/ZoomOut";
import FullscreenSharp from "@mui/icons-material/FullscreenSharp";
import FullscreenExitSharp from "@mui/icons-material/FullscreenExitSharp";
import ExitToApp from "@mui/icons-material/ExitToApp";
import CheckCircleOutline from "@mui/icons-material/CheckCircleOutline";
import RemoveCircleOutline from "@mui/icons-material/RemoveCircleOutline";
import NotInterested from "@mui/icons-material/NotInterested";
import Image from "@mui/icons-material/Image";
import Stop from "@mui/icons-material/Stop";
import FileCopy from "@mui/icons-material/FileCopy";
import ShowChart from "@mui/icons-material/ShowChart";

// Import FontAwesome icons
import { library } from "@fortawesome/fontawesome-svg-core";
import { faPencilAlt, faSlash, faUsers, faEllipsisH } from "@fortawesome/free-solid-svg-icons";
import { FormatAlignLeft } from "@mui/icons-material";
library.add(faPencilAlt, faSlash, faUsers, faEllipsisH);

export const FreezeIcon: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M20.79,13.95L18.46,14.57L16.46,13.44V10.56L18.46,9.43L20.79,10.05L21.31,8.12L19.54,7.65L20,5.88L18.07,5.36L17.45,7.69L15.45,8.82L13,7.38V5.12L14.71,3.41L13.29,2L12,3.29L10.71,2L9.29,3.41L11,5.12V7.38L8.5,8.82L6.5,7.69L5.92,5.36L4,5.88L4.47,7.65L2.7,8.12L3.22,10.05L5.55,9.43L7.55,10.56V13.45L5.55,14.58L3.22,13.96L2.7,15.89L4.47,16.36L4,18.12L5.93,18.64L6.55,16.31L8.55,15.18L11,16.62V18.88L9.29,20.59L10.71,22L12,20.71L13.29,22L14.7,20.59L13,18.88V16.62L15.5,15.17L17.5,16.3L18.12,18.63L20,18.12L19.53,16.35L21.3,15.88L20.79,13.95M9.5,10.56L12,9.11L14.5,10.56V13.44L12,14.89L9.5,13.44V10.56Z" />
    </SvgIcon>
);

export const UnfreezeIcon: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M11 5.12L9.29 3.41L10.71 2L12 3.29L13.29 2L14.71 3.41L13 5.12V7.38L15.45 8.82L17.45 7.69L18.07 5.36L20 5.88L19.54 7.65L21.31 8.12L20.79 10.05L18.46 9.43L16.46 10.56V13.26L14.5 11.3V10.56L12.74 9.54L10.73 7.53L11 7.38V5.12M18.46 14.57L16.87 13.67L19.55 16.35L21.3 15.88L20.79 13.95L18.46 14.57M13 16.62V18.88L14.7 20.59L13.29 22L12 20.71L10.71 22L9.29 20.59L11 18.88V16.62L8.55 15.18L6.55 16.31L5.93 18.64L4 18.12L4.47 16.36L2.7 15.89L3.22 13.96L5.55 14.58L7.55 13.45V10.56L5.55 9.43L3.22 10.05L2.7 8.12L4.47 7.65L4 5.89L1.11 3L2.39 1.73L22.11 21.46L20.84 22.73L14.1 16L13 16.62M12 14.89L12.63 14.5L9.5 11.39V13.44L12 14.89Z" />
    </SvgIcon>
);

export const PendingIcon = (props: any) => (
    <span {...props}>
        <span className="fas fa-ellipsis-h" />
    </span>
);

export const ReadOnlyIcon = (props: any) => (
    <span {...props}>
        <div className="fa-layers fa-1x fa-fw">
            <span
                className="fas fa-slash"
                data-fa-mask="fas fa-pencil-alt"
                data-fa-transform="down-1.5"
            />
            <span className="fas fa-slash" />
        </div>
    </span>
);

export const GroupsIcon = (props: any) => (
    <span {...props}>
        <span className="fas fa-users" />
    </span>
);

export const CollectionOldVersionIcon = (props: any) => (
    <Tooltip title="Old version">
        <Badge badgeContent={<History fontSize="small" />}>
            <CollectionIcon {...props} />
        </Badge>
    </Tooltip>
);

// https://pictogrammers.com/library/mdi/icon/ship-wheel/
export const WheelIcon = (props: any) => (
    <SvgIcon {...props} style={{fontSize: '1.6rem'}}>
        <path d="M2,11L4.05,11.1C4.3,8.83 5.5,6.85 7.25,5.56L6.13,3.84C5.86,3.36 6,2.75 6.5,2.47C7,2.2 7.59,2.36 7.87,2.84L8.8,4.66C9.78,4.24 10.86,4 12,4C13.14,4 14.22,4.24 15.2,4.66L16.13,2.84C16.41,2.36 17,2.2 17.5,2.47C18,2.75 18.14,3.36 17.87,3.84L16.75,5.56C18.5,6.85 19.7,8.83 19.95,11.1L22,11A1,1 0 0,1 23,12A1,1 0 0,1 22,13L19.95,12.9C19.7,15.17 18.5,17.15 16.75,18.44L17.87,20.16C18.14,20.64 18,21.25 17.5,21.53C17,21.8 16.41,21.64 16.13,21.16L15.2,19.34C14.22,19.76 13.14,20 12,20C10.86,20 9.78,19.76 8.8,19.34L7.87,21.16C7.59,21.64 7,21.8 6.5,21.53C6,21.25 5.86,20.64 6.13,20.16L7.25,18.44C5.5,17.15 4.3,15.17 4.05,12.9L2,13A1,1 0 0,1 1,12A1,1 0 0,1 2,11M9.07,11.35C9.2,10.74 9.53,10.2 10,9.79L8.34,7.25C7.11,8.19 6.27,9.6 6.05,11.2L9.07,11.35M12,9C12.32,9 12.62,9.05 12.9,9.14L14.28,6.45C13.58,6.16 12.81,6 12,6C11.19,6 10.42,6.16 9.72,6.45L11.1,9.14C11.38,9.05 11.68,9 12,9M14.93,11.35L17.95,11.2C17.73,9.6 16.89,8.19 15.66,7.25L14,9.79C14.47,10.2 14.8,10.74 14.93,11.35M14.93,12.65C14.8,13.26 14.47,13.8 14,14.21L15.66,16.75C16.89,15.81 17.73,14.4 17.95,12.8L14.93,12.65M12,15C11.68,15 11.38,14.95 11.09,14.86L9.72,17.55C10.42,17.84 11.19,18 12,18C12.81,18 13.58,17.84 14.28,17.55L12.91,14.86C12.62,14.95 12.32,15 12,15M9.07,12.65L6.05,12.8C6.27,14.4 7.11,15.81 8.34,16.75L10,14.21C9.53,13.8 9.2,13.26 9.07,12.65Z" />
    </SvgIcon>
);

// https://materialdesignicons.com/icon/image-off
export const ImageOffIcon = (props: any) => (
    <SvgIcon {...props}>
        <path d="M21 17.2L6.8 3H19C20.1 3 21 3.9 21 5V17.2M20.7 22L19.7 21H5C3.9 21 3 20.1 3 19V4.3L2 3.3L3.3 2L22 20.7L20.7 22M16.8 18L12.9 14.1L11 16.5L8.5 13.5L5 18H16.8Z" />
    </SvgIcon>
);

// https://materialdesignicons.com/icon/inbox-arrow-up
export const OutputIcon: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M14,14H10V11H8L12,7L16,11H14V14M16,11M5,15V5H19V15H15A3,3 0 0,1 12,18A3,3 0 0,1 9,15H5M19,3H5C3.89,3 3,3.9 3,5V19A2,2 0 0,0 5,21H19A2,2 0 0,0 21,19V5A2,2 0 0,0 19,3" />
    </SvgIcon>
);

// https://pictogrammers.com/library/mdi/icon/file-move/
export const FileMoveIcon: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M14,17H18V14L23,18.5L18,23V20H14V17M13,9H18.5L13,3.5V9M6,2H14L20,8V12.34C19.37,12.12 18.7,12 18,12A6,6 0 0,0 12,18C12,19.54 12.58,20.94 13.53,22H6C4.89,22 4,21.1 4,20V4A2,2 0 0,1 6,2Z" />
    </SvgIcon>
);

// https://pictogrammers.com/library/mdi/icon/checkbox-multiple-outline/
export const CheckboxMultipleOutline: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M20,2H8A2,2 0 0,0 6,4V16A2,2 0 0,0 8,18H20A2,2 0 0,0 22,16V4A2,2 0 0,0 20,2M20,16H8V4H20V16M16,20V22H4A2,2 0 0,1 2,20V7H4V20H16M18.53,8.06L17.47,7L12.59,11.88L10.47,9.76L9.41,10.82L12.59,14L18.53,8.06Z" />
    </SvgIcon>
);

// https://pictogrammers.com/library/mdi/icon/checkbox-multiple-blank-outline/
export const CheckboxMultipleBlankOutline: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M20,16V4H8V16H20M22,16A2,2 0 0,1 20,18H8C6.89,18 6,17.1 6,16V4C6,2.89 6.89,2 8,2H20A2,2 0 0,1 22,4V16M16,20V22H4A2,2 0 0,1 2,20V7H4V20H16Z" />
    </SvgIcon>
);

//https://pictogrammers.com/library/mdi/icon/console/
export const TerminalIcon: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M20,19V7H4V19H20M20,3A2,2 0 0,1 22,5V19A2,2 0 0,1 20,21H4A2,2 0 0,1 2,19V5C2,3.89 2.9,3 4,3H20M13,17V15H18V17H13M9.58,13L5.57,9H8.4L11.7,12.3C12.09,12.69 12.09,13.33 11.7,13.72L8.42,17H5.59L9.58,13Z" />
    </SvgIcon>
)

//https://pictogrammers.com/library/mdi/icon/chevron-double-right/
export const DoubleRightArrows: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M5.59,7.41L7,6L13,12L7,18L5.59,16.59L10.17,12L5.59,7.41M11.59,7.41L13,6L19,12L13,18L11.59,16.59L16.17,12L11.59,7.41Z" />
    </SvgIcon>
)

//https://pictogrammers.com/library/memory/icon/box-light-vertical/
export const VerticalLineDivider: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M12 0V22H10V0H12Z" />
    </SvgIcon>
)

//https://pictogrammers.com/library/mdi/icon/delete-forever/
export const DeleteForever: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19M8.46,11.88L9.87,10.47L12,12.59L14.12,10.47L15.53,11.88L13.41,14L15.53,16.12L14.12,17.53L12,15.41L9.88,17.53L8.47,16.12L10.59,14L8.46,11.88M15.5,4L14.5,3H9.5L8.5,4H5V6H19V4H15.5Z" />
    </SvgIcon>
)

//https://pictogrammers.com/library/mdi/icon/folder-key-outline/
export const FolderKeyIcon: IconType = (props: any) => (
    <SvgIcon {...props}>
        <path d="M20 18H4V8H20M20 6H12L10 4H4C2.9 4 2 4.9 2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6M12.8 12C12.4 10.8 11.3 10 10 10C8.3 10 7 11.3 7 13S8.3 16 10 16C11.3 16 12.4 15.2 12.8 14H15V16H17V14H19V12H12.8M10 14C9.4 14 9 13.6 9 13C9 12.4 9.4 12 10 12S11 12.4 11 13 10.6 14 10 14Z" />
    </SvgIcon>
)

export type IconType = React.SFC<{ className?: string; style?: object }>;

export const AddIcon: IconType = props => <Add {...props} />;
export const AddFavoriteIcon: IconType = props => <StarBorder {...props} />;
export const AdminMenuIcon: IconType = props => <Build {...props} />;
export const AdvancedIcon: IconType = props => <SettingsApplications {...props} />;
export const AttributesIcon: IconType = props => <ListAlt {...props} />;
export const BackIcon: IconType = props => <ArrowBack {...props} />;
export const CustomizeTableIcon: IconType = props => <Menu {...props} />;
export const CommandIcon: IconType = props => <LastPage {...props} />;
export const CopyIcon: IconType = props => <ContentCopy {...props} />;
export const FileCopyIcon: IconType = props => <FileCopy {...props} />;
export const FileCopyOutlinedIcon: IconType = props => <FileCopyOutlined {...props} />;
export const CollectionIcon: IconType = props => <LibraryBooks {...props} />;
export const CloseIcon: IconType = props => <Close {...props} />;
export const CloudUploadIcon: IconType = props => <CloudUpload {...props} />;
export const DefaultIcon: IconType = props => <RateReview {...props} />;
export const DetailsIcon: IconType = props => <Info {...props} />;
export const DirectoryIcon: IconType = props => <Folder {...props} />;
export const DownloadIcon: IconType = props => <GetApp {...props} />;
export const EditSavedQueryIcon: IconType = props => <Create {...props} />;
export const ExpandIcon: IconType = props => <ExpandMoreIcon {...props} />;
export const ErrorIcon: IconType = props => (
    <ErrorRoundedIcon
        style={{ color: "#ff0000" }}
        {...props}
    />
);
export const FavoriteIcon: IconType = props => <Star {...props} />;
export const FileIcon: IconType = props => <DescriptionIcon {...props} />;
export const HelpIcon: IconType = props => <Help {...props} />;
export const HelpOutlineIcon: IconType = props => <HelpOutline {...props} />;
export const ImportContactsIcon: IconType = props => <ImportContacts {...props} />;
export const InfoIcon: IconType = props => <Info {...props} />;
export const FileInputIcon: IconType = props => <InsertDriveFile {...props} />;
export const KeyIcon: IconType = props => <VpnKey {...props} />;
export const LogIcon: IconType = props => <SettingsEthernet {...props} />;
export const MailIcon: IconType = props => <Mail {...props} />;
export const MaximizeIcon: IconType = props => <FullscreenSharp {...props} />;
export const ResourceIcon: IconType = props => <Memory {...props} />;
export const UnMaximizeIcon: IconType = props => <FullscreenExitSharp {...props} />;
export const MoreVerticalIcon: IconType = props => <MoreVert {...props} />;
export const MoreHorizontalIcon: IconType = props => <MoreHoriz {...props} />;
export const MoveToIcon: IconType = props => <Input {...props} />;
export const NewProjectIcon: IconType = props => <CreateNewFolder {...props} />;
export const NotificationIcon: IconType = props => <Notifications {...props} />;
export const OpenIcon: IconType = props => <OpenInNew {...props} />;
export const InputIcon: IconType = props => <MoveToInbox {...props} />;
export const PaginationDownIcon: IconType = props => <ArrowDropDown {...props} />;
export const PaginationLeftArrowIcon: IconType = props => <ChevronLeft {...props} />;
export const PaginationRightArrowIcon: IconType = props => <ChevronRight {...props} />;
export const ProcessIcon: IconType = props => <Settings {...props} />;
export const ProjectIcon: IconType = props => <Folder {...props} />;
export const FilterGroupIcon: IconType = props => <Pageview {...props} />;
export const ProjectsIcon: IconType = props => <Inbox {...props} />;
export const ProvenanceGraphIcon: IconType = props => <DeviceHub {...props} />;
export const RemoveIcon: IconType = props => <Delete {...props} />;
export const RemoveFavoriteIcon: IconType = props => <Star {...props} />;
export const PublicFavoriteIcon: IconType = props => <Public {...props} />;
export const RenameIcon: IconType = props => <Edit {...props} />;
export const RestoreVersionIcon: IconType = props => <FlipToFront {...props} />;
export const RestoreFromTrashIcon: IconType = props => <RestoreFromTrash {...props} />;
export const ReRunProcessIcon: IconType = props => <Cached {...props} />;
export const SearchIcon: IconType = props => <Search {...props} />;
export const ShareIcon: IconType = props => <PersonAdd {...props} />;
export const ShareMeIcon: IconType = props => <People {...props} />;
export const SidePanelRightArrowIcon: IconType = props => <PlayArrow {...props} />;
export const TrashIcon: IconType = props => <Delete {...props} />;
export const UserPanelIcon: IconType = props => <Person {...props} />;
export const UsedByIcon: IconType = props => <Folder {...props} />;
export const WorkflowIcon: IconType = props => <Code {...props} />;
export const WarningIcon: IconType = props => (
    <Warning
        style={{ color: "#fbc02d", height: "30px", width: "30px" }}
        {...props}
    />
);
export const Link: IconType = props => <LinkOutlined {...props} />;
export const FolderSharedIcon: IconType = props => <FolderShared {...props} />;
export const CanReadIcon: IconType = props => <RemoveRedEye {...props} />;
export const CanWriteIcon: IconType = props => <Edit {...props} />;
export const CanManageIcon: IconType = props => <Computer {...props} />;
export const AddUserIcon: IconType = props => <PersonAdd {...props} />;
export const WordWrapOnIcon: IconType = props => <WrapText {...props} />;
export const WordWrapOffIcon: IconType = props => <FormatAlignLeft {...props} />;
export const TextIncreaseIcon: IconType = props => <TextIncrease {...props} />;
export const TextDecreaseIcon: IconType = props => <TextDecrease {...props} />;
export const DeactivateUserIcon: IconType = props => <NotInterested {...props} />;
export const LoginAsIcon: IconType = props => <ExitToApp {...props} />;
export const ActiveIcon: IconType = props => <CheckCircleOutline {...props} />;
export const SetupIcon: IconType = props => <RemoveCircleOutline {...props} />;
export const InactiveIcon: IconType = props => <NotInterested {...props} />;
export const ImageIcon: IconType = props => <Image {...props} />;
export const StartIcon: IconType = props => <PlayArrow {...props} />;
export const StopIcon: IconType = props => <Stop {...props} />;
export const SelectAllIcon: IconType = props => <CheckboxMultipleOutline {...props} />;
export const SelectNoneIcon: IconType = props => <CheckboxMultipleBlankOutline {...props} />;
export const ShowChartIcon: IconType = props => <ShowChart {...props} />;
