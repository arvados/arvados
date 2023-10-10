// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

<<<<<<< HEAD
import React from 'react';
import { Grid, Typography, withStyles, Tooltip, IconButton, Checkbox, Chip } from '@material-ui/core';
import { FavoriteStar, PublicFavoriteStar } from '../favorite-star/favorite-star';
import { Resource, ResourceKind, TrashableResource } from 'models/resource';
=======
import React from "react";
import { Grid, Typography, withStyles, Tooltip, IconButton, Checkbox, Chip } from "@material-ui/core";
import { FavoriteStar, PublicFavoriteStar } from "../favorite-star/favorite-star";
import { Resource, ResourceKind, TrashableResource } from "models/resource";
>>>>>>> main
import {
    FreezeIcon,
    ProjectIcon,
    FilterGroupIcon,
    CollectionIcon,
    ProcessIcon,
    DefaultIcon,
    ShareIcon,
    CollectionOldVersionIcon,
    WorkflowIcon,
    RemoveIcon,
    RenameIcon,
    ActiveIcon,
    SetupIcon,
    InactiveIcon,
<<<<<<< HEAD
} from 'components/icon/icon';
import { formatDate, formatFileSize, formatTime } from 'common/formatters';
import { resourceLabel } from 'common/labels';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { getResource, filterResources } from 'store/resources/resources';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { getProcess, Process, getProcessStatus, getProcessStatusStyles, getProcessRuntime } from 'store/processes/process';
import { ArvadosTheme } from 'common/custom-theme';
import { compose, Dispatch } from 'redux';
import { WorkflowResource } from 'models/workflow';
import { ResourceStatus as WorkflowStatus } from 'views/workflow-panel/workflow-panel-view';
import { getUuidPrefix, openRunProcess } from 'store/workflow-panel/workflow-panel-actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';
import { getUserFullname, getUserDisplayName, User, UserResource } from 'models/user';
import { toggleIsAdmin } from 'store/users/users-actions';
import { LinkClass, LinkResource } from 'models/link';
import { navigateTo, navigateToGroupDetails, navigateToUserProfile } from 'store/navigation/navigation-action';
import { withResourceData } from 'views-components/data-explorer/with-resources';
import { CollectionResource } from 'models/collection';
import { IllegalNamingWarning } from 'components/warning/warning';
import { loadResource } from 'store/resources/resources-actions';
import { BuiltinGroups, getBuiltinGroupUuid, GroupClass, GroupResource, isBuiltinGroup } from 'models/group';
import { openRemoveGroupMemberDialog } from 'store/group-details-panel/group-details-panel-actions';
import { setMemberIsHidden } from 'store/group-details-panel/group-details-panel-actions';
import { formatPermissionLevel } from 'views-components/sharing-dialog/permission-select';
import { PermissionLevel } from 'models/permission';
import { openPermissionEditContextMenu } from 'store/context-menu/context-menu-actions';
import { getUserUuid } from 'common/getuser';
import { VirtualMachinesResource } from 'models/virtual-machines';
import { CopyToClipboardSnackbar } from 'components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar';
import { ProjectResource } from 'models/project';
import { ProcessResource } from 'models/process';

const renderName = (dispatch: Dispatch, item: GroupContentsResource) => {
    const navFunc = 'groupClass' in item && item.groupClass === GroupClass.ROLE ? navigateToGroupDetails : navigateTo;
    return (
        <Grid container alignItems='center' wrap='nowrap' spacing={16}>
            <Grid item>{renderIcon(item)}</Grid>
            <Grid item>
                <Typography color='primary' style={{ width: 'auto', cursor: 'pointer' }} onClick={() => dispatch<any>(navFunc(item.uuid))}>
=======
} from "components/icon/icon";
import { formatDate, formatFileSize, formatTime } from "common/formatters";
import { resourceLabel } from "common/labels";
import { connect, DispatchProp } from "react-redux";
import { RootState } from "store/store";
import { getResource, filterResources } from "store/resources/resources";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { getProcess, Process, getProcessStatus, getProcessStatusStyles, getProcessRuntime } from "store/processes/process";
import { ArvadosTheme } from "common/custom-theme";
import { compose, Dispatch } from "redux";
import { WorkflowResource } from "models/workflow";
import { ResourceStatus as WorkflowStatus } from "views/workflow-panel/workflow-panel-view";
import { getUuidPrefix, openRunProcess } from "store/workflow-panel/workflow-panel-actions";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { getUserFullname, getUserDisplayName, User, UserResource } from "models/user";
import { toggleIsAdmin } from "store/users/users-actions";
import { LinkClass, LinkResource } from "models/link";
import { navigateTo, navigateToGroupDetails, navigateToUserProfile } from "store/navigation/navigation-action";
import { withResourceData } from "views-components/data-explorer/with-resources";
import { CollectionResource } from "models/collection";
import { IllegalNamingWarning } from "components/warning/warning";
import { loadResource } from "store/resources/resources-actions";
import { BuiltinGroups, getBuiltinGroupUuid, GroupClass, GroupResource, isBuiltinGroup } from "models/group";
import { openRemoveGroupMemberDialog } from "store/group-details-panel/group-details-panel-actions";
import { setMemberIsHidden } from "store/group-details-panel/group-details-panel-actions";
import { formatPermissionLevel } from "views-components/sharing-dialog/permission-select";
import { PermissionLevel } from "models/permission";
import { openPermissionEditContextMenu } from "store/context-menu/context-menu-actions";
import { VirtualMachinesResource } from "models/virtual-machines";
import { CopyToClipboardSnackbar } from "components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar";
import { ProjectResource } from "models/project";
import { ProcessResource } from "models/process";

const renderName = (dispatch: Dispatch, item: GroupContentsResource) => {
    const navFunc = "groupClass" in item && item.groupClass === GroupClass.ROLE ? navigateToGroupDetails : navigateTo;
    return (
        <Grid
            container
            alignItems="center"
            wrap="nowrap"
            spacing={16}
        >
            <Grid item>{renderIcon(item)}</Grid>
            <Grid item>
                <Typography
                    color="primary"
                    style={{ width: "auto", cursor: "pointer" }}
                    onClick={() => dispatch<any>(navFunc(item.uuid))}
                >
>>>>>>> main
                    {item.kind === ResourceKind.PROJECT || item.kind === ResourceKind.COLLECTION ? <IllegalNamingWarning name={item.name} /> : null}
                    {item.name}
                </Typography>
            </Grid>
            <Grid item>
<<<<<<< HEAD
                <Typography variant='caption'>
=======
                <Typography variant="caption">
>>>>>>> main
                    <FavoriteStar resourceUuid={item.uuid} />
                    <PublicFavoriteStar resourceUuid={item.uuid} />
                    {item.kind === ResourceKind.PROJECT && <FrozenProject item={item} />}
                </Typography>
            </Grid>
        </Grid>
    );
};

const FrozenProject = (props: { item: ProjectResource }) => {
    const [fullUsername, setFullusername] = React.useState<any>(null);
    const getFullName = React.useCallback(() => {
        if (props.item.frozenByUuid) {
            setFullusername(<UserNameFromID uuid={props.item.frozenByUuid} />);
        }
    }, [props.item, setFullusername]);

    if (props.item.frozenByUuid) {
        return (
<<<<<<< HEAD
            <Tooltip onOpen={getFullName} enterDelay={500} title={<span>Project was frozen by {fullUsername}</span>}>
                <FreezeIcon style={{ fontSize: 'inherit' }} />
=======
            <Tooltip
                onOpen={getFullName}
                enterDelay={500}
                title={<span>Project was frozen by {fullUsername}</span>}
            >
                <FreezeIcon style={{ fontSize: "inherit" }} />
>>>>>>> main
            </Tooltip>
        );
    } else {
        return null;
    }
};

export const ResourceName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
    return resource;
})((resource: GroupContentsResource & DispatchProp<any>) => renderName(resource.dispatch, resource));

const renderIcon = (item: GroupContentsResource) => {
    switch (item.kind) {
        case ResourceKind.PROJECT:
            if (item.groupClass === GroupClass.FILTER) {
                return <FilterGroupIcon />;
            }
            return <ProjectIcon />;
        case ResourceKind.COLLECTION:
            if (item.uuid === item.currentVersionUuid) {
                return <CollectionIcon />;
            }
            return <CollectionOldVersionIcon />;
        case ResourceKind.PROCESS:
            return <ProcessIcon />;
        case ResourceKind.WORKFLOW:
            return <WorkflowIcon />;
        default:
            return <DefaultIcon />;
    }
};

const renderDate = (date?: string) => {
    return (
<<<<<<< HEAD
        <Typography noWrap style={{ minWidth: '100px' }}>
=======
        <Typography
            noWrap
            style={{ minWidth: "100px" }}
        >
>>>>>>> main
            {formatDate(date)}
        </Typography>
    );
};

const renderWorkflowName = (item: WorkflowResource) => (
<<<<<<< HEAD
    <Grid container alignItems='center' wrap='nowrap' spacing={16}>
        <Grid item>{renderIcon(item)}</Grid>
        <Grid item>
            <Typography color='primary' style={{ width: '100px' }}>
=======
    <Grid
        container
        alignItems="center"
        wrap="nowrap"
        spacing={16}
    >
        <Grid item>{renderIcon(item)}</Grid>
        <Grid item>
            <Typography
                color="primary"
                style={{ width: "100px" }}
            >
>>>>>>> main
                {item.name}
            </Typography>
        </Grid>
    </Grid>
);

export const ResourceWorkflowName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
    return resource;
})(renderWorkflowName);

const getPublicUuid = (uuidPrefix: string) => {
    return `${uuidPrefix}-tpzed-anonymouspublic`;
};

const resourceShare = (dispatch: Dispatch, uuidPrefix: string, ownerUuid?: string, uuid?: string) => {
    const isPublic = ownerUuid === getPublicUuid(uuidPrefix);
    return (
        <div>
            {!isPublic && uuid && (
<<<<<<< HEAD
                <Tooltip title='Share'>
=======
                <Tooltip title="Share">
>>>>>>> main
                    <IconButton onClick={() => dispatch<any>(openSharingDialog(uuid))}>
                        <ShareIcon />
                    </IconButton>
                </Tooltip>
            )}
        </div>
    );
};

export const ResourceShare = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
    const uuidPrefix = getUuidPrefix(state);
    return {
<<<<<<< HEAD
        uuid: resource ? resource.uuid : '',
        ownerUuid: resource ? resource.ownerUuid : '',
=======
        uuid: resource ? resource.uuid : "",
        ownerUuid: resource ? resource.ownerUuid : "",
>>>>>>> main
        uuidPrefix,
    };
})((props: { ownerUuid?: string; uuidPrefix: string; uuid?: string } & DispatchProp<any>) =>
    resourceShare(props.dispatch, props.uuidPrefix, props.ownerUuid, props.uuid)
);

// User Resources
const renderFirstName = (item: { firstName: string }) => {
    return <Typography noWrap>{item.firstName}</Typography>;
};

export const ResourceFirstName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<UserResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { firstName: '' };
=======
    return resource || { firstName: "" };
>>>>>>> main
})(renderFirstName);

const renderLastName = (item: { lastName: string }) => <Typography noWrap>{item.lastName}</Typography>;

export const ResourceLastName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<UserResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { lastName: '' };
})(renderLastName);

const renderFullName = (dispatch: Dispatch, item: { uuid: string; firstName: string; lastName: string }, link?: boolean) => {
    const displayName = (item.firstName + ' ' + item.lastName).trim() || item.uuid;
    return link ? (
        <Typography noWrap color='primary' style={{ cursor: 'pointer' }} onClick={() => dispatch<any>(navigateToUserProfile(item.uuid))}>
=======
    return resource || { lastName: "" };
})(renderLastName);

const renderFullName = (dispatch: Dispatch, item: { uuid: string; firstName: string; lastName: string }, link?: boolean) => {
    const displayName = (item.firstName + " " + item.lastName).trim() || item.uuid;
    return link ? (
        <Typography
            noWrap
            color="primary"
            style={{ cursor: "pointer" }}
            onClick={() => dispatch<any>(navigateToUserProfile(item.uuid))}
        >
>>>>>>> main
            {displayName}
        </Typography>
    ) : (
        <Typography noWrap>{displayName}</Typography>
    );
};

export const UserResourceFullName = connect((state: RootState, props: { uuid: string; link?: boolean }) => {
    const resource = getResource<UserResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { item: resource || { uuid: '', firstName: '', lastName: '' }, link: props.link };
=======
    return { item: resource || { uuid: "", firstName: "", lastName: "" }, link: props.link };
>>>>>>> main
})((props: { item: { uuid: string; firstName: string; lastName: string }; link?: boolean } & DispatchProp<any>) =>
    renderFullName(props.dispatch, props.item, props.link)
);

const renderUuid = (item: { uuid: string }) => (
<<<<<<< HEAD
    <Typography data-cy='uuid' noWrap>
        {item.uuid}
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || '-'}
=======
    <Typography
        data-cy="uuid"
        noWrap
    >
        {item.uuid}
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || "-"}
>>>>>>> main
    </Typography>
);

const renderUuidCopyIcon = (item: { uuid: string }) => (
<<<<<<< HEAD
    <Typography data-cy='uuid' noWrap>
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || '-'}
    </Typography>
);

export const ResourceUuid = connect((state: RootState, props: { uuid: string }) => getResource<UserResource>(props.uuid)(state.resources) || { uuid: '' })(renderUuid);
=======
    <Typography
        data-cy="uuid"
        noWrap
    >
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || "-"}
    </Typography>
);

export const ResourceUuid = connect(
    (state: RootState, props: { uuid: string }) => getResource<UserResource>(props.uuid)(state.resources) || { uuid: "" }
)(renderUuid);
>>>>>>> main

const renderEmail = (item: { email: string }) => <Typography noWrap>{item.email}</Typography>;

export const ResourceEmail = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<UserResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { email: '' };
})(renderEmail);

enum UserAccountStatus {
    ACTIVE = 'Active',
    INACTIVE = 'Inactive',
    SETUP = 'Setup',
    UNKNOWN = '',
}

const renderAccountStatus = (props: { status: UserAccountStatus }) => (
    <Grid container alignItems='center' wrap='nowrap' spacing={8} data-cy='account-status'>
=======
    return resource || { email: "" };
})(renderEmail);

enum UserAccountStatus {
    ACTIVE = "Active",
    INACTIVE = "Inactive",
    SETUP = "Setup",
    UNKNOWN = "",
}

const renderAccountStatus = (props: { status: UserAccountStatus }) => (
    <Grid
        container
        alignItems="center"
        wrap="nowrap"
        spacing={8}
        data-cy="account-status"
    >
>>>>>>> main
        <Grid item>
            {(() => {
                switch (props.status) {
                    case UserAccountStatus.ACTIVE:
<<<<<<< HEAD
                        return <ActiveIcon style={{ color: '#4caf50', verticalAlign: 'middle' }} />;
                    case UserAccountStatus.SETUP:
                        return <SetupIcon style={{ color: '#2196f3', verticalAlign: 'middle' }} />;
                    case UserAccountStatus.INACTIVE:
                        return <InactiveIcon style={{ color: '#9e9e9e', verticalAlign: 'middle' }} />;
=======
                        return <ActiveIcon style={{ color: "#4caf50", verticalAlign: "middle" }} />;
                    case UserAccountStatus.SETUP:
                        return <SetupIcon style={{ color: "#2196f3", verticalAlign: "middle" }} />;
                    case UserAccountStatus.INACTIVE:
                        return <InactiveIcon style={{ color: "#9e9e9e", verticalAlign: "middle" }} />;
>>>>>>> main
                    default:
                        return <></>;
                }
            })()}
        </Grid>
        <Grid item>
            <Typography noWrap>{props.status}</Typography>
        </Grid>
    </Grid>
);

const getUserAccountStatus = (state: RootState, props: { uuid: string }) => {
    const user = getResource<UserResource>(props.uuid)(state.resources);
    // Get membership links for all users group
    const allUsersGroupUuid = getBuiltinGroupUuid(state.auth.localCluster, BuiltinGroups.ALL);
    const permissions = filterResources(
        (resource: LinkResource) =>
            resource.kind === ResourceKind.LINK &&
            resource.linkClass === LinkClass.PERMISSION &&
            resource.headUuid === allUsersGroupUuid &&
            resource.tailUuid === props.uuid
    )(state.resources);

    if (user) {
        return user.isActive
            ? { status: UserAccountStatus.ACTIVE }
            : permissions.length > 0
            ? { status: UserAccountStatus.SETUP }
            : { status: UserAccountStatus.INACTIVE };
    } else {
        return { status: UserAccountStatus.UNKNOWN };
    }
};

export const ResourceLinkTailAccountStatus = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
    return link && link.tailKind === ResourceKind.USER ? getUserAccountStatus(state, { uuid: link.tailUuid }) : { status: UserAccountStatus.UNKNOWN };
})(renderAccountStatus);

export const UserResourceAccountStatus = connect(getUserAccountStatus)(renderAccountStatus);

const renderIsHidden = (props: {
    memberLinkUuid: string;
    permissionLinkUuid: string;
    visible: boolean;
    canManage: boolean;
    setMemberIsHidden: (memberLinkUuid: string, permissionLinkUuid: string, hide: boolean) => void;
}) => {
    if (props.memberLinkUuid) {
        return (
            <Checkbox
<<<<<<< HEAD
                data-cy='user-visible-checkbox'
                color='primary'
                checked={props.visible}
                disabled={!props.canManage}
                onClick={(e) => {
=======
                data-cy="user-visible-checkbox"
                color="primary"
                checked={props.visible}
                disabled={!props.canManage}
                onClick={e => {
>>>>>>> main
                    e.stopPropagation();
                    props.setMemberIsHidden(props.memberLinkUuid, props.permissionLinkUuid, !props.visible);
                }}
            />
        );
    } else {
        return <Typography />;
    }
};

export const ResourceLinkTailIsVisible = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const member = getResource<Resource>(link?.tailUuid || "")(state.resources);
        const group = getResource<GroupResource>(link?.headUuid || "")(state.resources);
        const permissions = filterResources((resource: LinkResource) => {
            return (
                resource.linkClass === LinkClass.PERMISSION &&
                resource.headUuid === link?.tailUuid &&
                resource.tailUuid === group?.uuid &&
                resource.name === PermissionLevel.CAN_READ
            );
        })(state.resources);

        const permissionLinkUuid = permissions.length > 0 ? permissions[0].uuid : "";
        const isVisible = link && group && permissions.length > 0;
        // Consider whether the current user canManage this resurce in addition when it's possible
        const isBuiltin = isBuiltinGroup(link?.headUuid || "");

        return member?.kind === ResourceKind.USER
            ? { memberLinkUuid: link?.uuid, permissionLinkUuid, visible: isVisible, canManage: !isBuiltin }
<<<<<<< HEAD
            : { memberLinkUuid: '', permissionLinkUuid: '', visible: false, canManage: false };
=======
            : { memberLinkUuid: "", permissionLinkUuid: "", visible: false, canManage: false };
>>>>>>> main
    },
    { setMemberIsHidden }
)(renderIsHidden);

const renderIsAdmin = (props: { uuid: string; isAdmin: boolean; toggleIsAdmin: (uuid: string) => void }) => (
    <Checkbox
        color='primary'
        checked={props.isAdmin}
        onClick={e => {
            e.stopPropagation();
            props.toggleIsAdmin(props.uuid);
        }}
    />
);

export const ResourceIsAdmin = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { isAdmin: false };
    },
    { toggleIsAdmin }
)(renderIsAdmin);

const renderUsername = (item: { username: string; uuid: string }) => <Typography noWrap>{item.username || item.uuid}</Typography>;

export const ResourceUsername = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<UserResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { username: '', uuid: props.uuid };
=======
    return resource || { username: "", uuid: props.uuid };
>>>>>>> main
})(renderUsername);

// Virtual machine resource

const renderHostname = (item: { hostname: string }) => <Typography noWrap>{item.hostname}</Typography>;

export const VirtualMachineHostname = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<VirtualMachinesResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { hostname: '' };
=======
    return resource || { hostname: "" };
>>>>>>> main
})(renderHostname);

const renderVirtualMachineLogin = (login: { user: string }) => <Typography noWrap>{login.user}</Typography>;

export const VirtualMachineLogin = connect((state: RootState, props: { linkUuid: string }) => {
    const permission = getResource<LinkResource>(props.linkUuid)(state.resources);
<<<<<<< HEAD
    const user = getResource<UserResource>(permission?.tailUuid || '')(state.resources);

    return { user: user?.username || permission?.tailUuid || '' };
=======
    const user = getResource<UserResource>(permission?.tailUuid || "")(state.resources);

    return { user: user?.username || permission?.tailUuid || "" };
>>>>>>> main
})(renderVirtualMachineLogin);

// Common methods
const renderCommonData = (data: string) => <Typography noWrap>{data}</Typography>;

const renderCommonDate = (date: string) => <Typography noWrap>{formatDate(date)}</Typography>;

export const CommonUuid = withResourceData("uuid", renderCommonData);

// Api Client Authorizations
export const TokenApiClientId = withResourceData("apiClientId", renderCommonData);

export const TokenApiToken = withResourceData("apiToken", renderCommonData);

export const TokenCreatedByIpAddress = withResourceData("createdByIpAddress", renderCommonDate);

export const TokenDefaultOwnerUuid = withResourceData("defaultOwnerUuid", renderCommonData);

export const TokenExpiresAt = withResourceData("expiresAt", renderCommonDate);

export const TokenLastUsedAt = withResourceData("lastUsedAt", renderCommonDate);

export const TokenLastUsedByIpAddress = withResourceData("lastUsedByIpAddress", renderCommonData);

export const TokenScopes = withResourceData("scopes", renderCommonData);

export const TokenUserId = withResourceData("userId", renderCommonData);

const clusterColors = [
<<<<<<< HEAD
    ['#f44336', '#fff'],
    ['#2196f3', '#fff'],
    ['#009688', '#fff'],
    ['#cddc39', '#fff'],
    ['#ff9800', '#fff'],
=======
    ["#f44336", "#fff"],
    ["#2196f3", "#fff"],
    ["#009688", "#fff"],
    ["#cddc39", "#fff"],
    ["#ff9800", "#fff"],
>>>>>>> main
];

export const ResourceCluster = (props: { uuid: string }) => {
    const CLUSTER_ID_LENGTH = 5;
<<<<<<< HEAD
    const pos = props.uuid.length > CLUSTER_ID_LENGTH ? props.uuid.indexOf('-') : 5;
    const clusterId = pos >= CLUSTER_ID_LENGTH ? props.uuid.substring(0, pos) : '';
    const ci =
        pos >= CLUSTER_ID_LENGTH
            ? ((props.uuid.charCodeAt(0) * props.uuid.charCodeAt(1) + props.uuid.charCodeAt(2)) * props.uuid.charCodeAt(3) + props.uuid.charCodeAt(4)) %
=======
    const pos = props.uuid.length > CLUSTER_ID_LENGTH ? props.uuid.indexOf("-") : 5;
    const clusterId = pos >= CLUSTER_ID_LENGTH ? props.uuid.substring(0, pos) : "";
    const ci =
        pos >= CLUSTER_ID_LENGTH
            ? ((props.uuid.charCodeAt(0) * props.uuid.charCodeAt(1) + props.uuid.charCodeAt(2)) * props.uuid.charCodeAt(3) +
                  props.uuid.charCodeAt(4)) %
>>>>>>> main
              clusterColors.length
            : 0;
    return (
        <span
            style={{
                backgroundColor: clusterColors[ci][0],
                color: clusterColors[ci][1],
<<<<<<< HEAD
                padding: '2px 7px',
=======
                padding: "2px 7px",
>>>>>>> main
                borderRadius: 3,
            }}
        >
            {clusterId}
        </span>
    );
};

// Links Resources
<<<<<<< HEAD
const renderLinkName = (item: { name: string }) => <Typography noWrap>{item.name || '-'}</Typography>;

export const ResourceLinkName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<LinkResource>(props.uuid)(state.resources);
    return resource || { name: '' };
=======
const renderLinkName = (item: { name: string }) => <Typography noWrap>{item.name || "-"}</Typography>;

export const ResourceLinkName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<LinkResource>(props.uuid)(state.resources);
    return resource || { name: "" };
>>>>>>> main
})(renderLinkName);

const renderLinkClass = (item: { linkClass: string }) => <Typography noWrap>{item.linkClass}</Typography>;

export const ResourceLinkClass = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { linkClass: '' };
})(renderLinkClass);

const getResourceDisplayName = (resource: Resource): string => {
    if ((resource as UserResource).kind === ResourceKind.USER && typeof (resource as UserResource).firstName !== 'undefined') {
=======
    return resource || { linkClass: "" };
})(renderLinkClass);

const getResourceDisplayName = (resource: Resource): string => {
    if ((resource as UserResource).kind === ResourceKind.USER && typeof (resource as UserResource).firstName !== "undefined") {
>>>>>>> main
        // We can be sure the resource is UserResource
        return getUserDisplayName(resource as UserResource);
    } else {
        return (resource as GroupContentsResource).name;
    }
};

const renderResourceLink = (dispatch: Dispatch, item: Resource) => {
    var displayName = getResourceDisplayName(item);

    return (
<<<<<<< HEAD
        <Typography noWrap color='primary' style={{ cursor: 'pointer' }} onClick={() => dispatch<any>(navigateTo(item.uuid))}>
            {resourceLabel(item.kind, item && item.kind === ResourceKind.GROUP ? (item as GroupResource).groupClass || '' : '')}: {displayName || item.uuid}
=======
        <Typography
            noWrap
            color="primary"
            style={{ cursor: "pointer" }}
            onClick={() => {
                console.log(item);
                item.kind === ResourceKind.GROUP && (item as GroupResource).groupClass === "role"
                    ? dispatch<any>(navigateToGroupDetails(item.uuid))
                    : dispatch<any>(navigateTo(item.uuid));
            }}
        >
            {resourceLabel(item.kind, item && item.kind === ResourceKind.GROUP ? (item as GroupResource).groupClass || "" : "")}:{" "}
            {displayName || item.uuid}
>>>>>>> main
        </Typography>
    );
};

export const ResourceLinkTail = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const tailResource = getResource<Resource>(resource?.tailUuid || '')(state.resources);

    return {
        item: tailResource || { uuid: resource?.tailUuid || '', kind: resource?.tailKind || ResourceKind.NONE },
=======
    const tailResource = getResource<Resource>(resource?.tailUuid || "")(state.resources);

    return {
        item: tailResource || { uuid: resource?.tailUuid || "", kind: resource?.tailKind || ResourceKind.NONE },
>>>>>>> main
    };
})((props: { item: Resource } & DispatchProp<any>) => renderResourceLink(props.dispatch, props.item));

export const ResourceLinkHead = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const headResource = getResource<Resource>(resource?.headUuid || '')(state.resources);

    return {
        item: headResource || { uuid: resource?.headUuid || '', kind: resource?.headKind || ResourceKind.NONE },
=======
    const headResource = getResource<Resource>(resource?.headUuid || "")(state.resources);

    return {
        item: headResource || { uuid: resource?.headUuid || "", kind: resource?.headKind || ResourceKind.NONE },
>>>>>>> main
    };
})((props: { item: Resource } & DispatchProp<any>) => renderResourceLink(props.dispatch, props.item));

export const ResourceLinkUuid = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return resource || { uuid: '' };
=======
    return resource || { uuid: "" };
>>>>>>> main
})(renderUuid);

export const ResourceLinkHeadUuid = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const headResource = getResource<Resource>(link?.headUuid || '')(state.resources);

    return headResource || { uuid: '' };
=======
    const headResource = getResource<Resource>(link?.headUuid || "")(state.resources);

    return headResource || { uuid: "" };
>>>>>>> main
})(renderUuid);

export const ResourceLinkTailUuid = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const tailResource = getResource<Resource>(link?.tailUuid || '')(state.resources);

    return tailResource || { uuid: '' };
=======
    const tailResource = getResource<Resource>(link?.tailUuid || "")(state.resources);

    return tailResource || { uuid: "" };
>>>>>>> main
})(renderUuid);

const renderLinkDelete = (dispatch: Dispatch, item: LinkResource, canManage: boolean) => {
    if (item.uuid) {
        return canManage ? (
            <Typography noWrap>
<<<<<<< HEAD
                <IconButton data-cy='resource-delete-button' onClick={() => dispatch<any>(openRemoveGroupMemberDialog(item.uuid))}>
=======
                <IconButton
                    data-cy="resource-delete-button"
                    onClick={() => dispatch<any>(openRemoveGroupMemberDialog(item.uuid))}
                >
>>>>>>> main
                    <RemoveIcon />
                </IconButton>
            </Typography>
        ) : (
            <Typography noWrap>
<<<<<<< HEAD
                <IconButton disabled data-cy='resource-delete-button'>
=======
                <IconButton
                    disabled
                    data-cy="resource-delete-button"
                >
>>>>>>> main
                    <RemoveIcon />
                </IconButton>
            </Typography>
        );
    } else {
        return <Typography noWrap></Typography>;
    }
};

export const ResourceLinkDelete = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const isBuiltin = isBuiltinGroup(link?.headUuid || '') || isBuiltinGroup(link?.tailUuid || '');

    return {
        item: link || { uuid: '', kind: ResourceKind.NONE },
=======
    const isBuiltin = isBuiltinGroup(link?.headUuid || "") || isBuiltinGroup(link?.tailUuid || "");

    return {
        item: link || { uuid: "", kind: ResourceKind.NONE },
>>>>>>> main
        canManage: link && getResourceLinkCanManage(state, link) && !isBuiltin,
    };
})((props: { item: LinkResource; canManage: boolean } & DispatchProp<any>) => renderLinkDelete(props.dispatch, props.item, props.canManage));

export const ResourceLinkTailEmail = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const resource = getResource<UserResource>(link?.tailUuid || '')(state.resources);

    return resource || { email: '' };
=======
    const resource = getResource<UserResource>(link?.tailUuid || "")(state.resources);

    return resource || { email: "" };
>>>>>>> main
})(renderEmail);

export const ResourceLinkTailUsername = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const resource = getResource<UserResource>(link?.tailUuid || '')(state.resources);

    return resource || { username: '' };
=======
    const resource = getResource<UserResource>(link?.tailUuid || "")(state.resources);

    return resource || { username: "" };
>>>>>>> main
})(renderUsername);

const renderPermissionLevel = (dispatch: Dispatch, link: LinkResource, canManage: boolean) => {
    return (
        <Typography noWrap>
            {formatPermissionLevel(link.name as PermissionLevel)}
            {canManage ? (
<<<<<<< HEAD
                <IconButton data-cy='edit-permission-button' onClick={(event) => dispatch<any>(openPermissionEditContextMenu(event, link))}>
                    <RenameIcon />
                </IconButton>
            ) : (
                ''
=======
                <IconButton
                    data-cy="edit-permission-button"
                    onClick={event => dispatch<any>(openPermissionEditContextMenu(event, link))}
                >
                    <RenameIcon />
                </IconButton>
            ) : (
                ""
>>>>>>> main
            )}
        </Typography>
    );
};

export const ResourceLinkHeadPermissionLevel = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const isBuiltin = isBuiltinGroup(link?.headUuid || '') || isBuiltinGroup(link?.tailUuid || '');

    return {
        link: link || { uuid: '', name: '', kind: ResourceKind.NONE },
=======
    const isBuiltin = isBuiltinGroup(link?.headUuid || "") || isBuiltinGroup(link?.tailUuid || "");

    return {
        link: link || { uuid: "", name: "", kind: ResourceKind.NONE },
>>>>>>> main
        canManage: link && getResourceLinkCanManage(state, link) && !isBuiltin,
    };
})((props: { link: LinkResource; canManage: boolean } & DispatchProp<any>) => renderPermissionLevel(props.dispatch, props.link, props.canManage));

export const ResourceLinkTailPermissionLevel = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    const isBuiltin = isBuiltinGroup(link?.headUuid || '') || isBuiltinGroup(link?.tailUuid || '');

    return {
        link: link || { uuid: '', name: '', kind: ResourceKind.NONE },
=======
    const isBuiltin = isBuiltinGroup(link?.headUuid || "") || isBuiltinGroup(link?.tailUuid || "");

    return {
        link: link || { uuid: "", name: "", kind: ResourceKind.NONE },
>>>>>>> main
        canManage: link && getResourceLinkCanManage(state, link) && !isBuiltin,
    };
})((props: { link: LinkResource; canManage: boolean } & DispatchProp<any>) => renderPermissionLevel(props.dispatch, props.link, props.canManage));

const getResourceLinkCanManage = (state: RootState, link: LinkResource) => {
    const headResource = getResource<Resource>(link.headUuid)(state.resources);
    if (headResource && headResource.kind === ResourceKind.GROUP) {
        return (headResource as GroupResource).canManage;
    } else {
        // true for now
        return true;
    }
};

// Process Resources
const resourceRunProcess = (dispatch: Dispatch, uuid: string) => {
    return (
        <div>
            {uuid && (
<<<<<<< HEAD
                <Tooltip title='Run process'>
=======
                <Tooltip title="Run process">
>>>>>>> main
                    <IconButton onClick={() => dispatch<any>(openRunProcess(uuid))}>
                        <ProcessIcon />
                    </IconButton>
                </Tooltip>
            )}
        </div>
    );
};

export const ResourceRunProcess = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
    return {
<<<<<<< HEAD
        uuid: resource ? resource.uuid : '',
=======
        uuid: resource ? resource.uuid : "",
>>>>>>> main
    };
})((props: { uuid: string } & DispatchProp<any>) => resourceRunProcess(props.dispatch, props.uuid));

const renderWorkflowStatus = (uuidPrefix: string, ownerUuid?: string) => {
    if (ownerUuid === getPublicUuid(uuidPrefix)) {
        return renderStatus(WorkflowStatus.PUBLIC);
    } else {
        return renderStatus(WorkflowStatus.PRIVATE);
    }
};

const renderStatus = (status: string) => (
<<<<<<< HEAD
    <Typography noWrap style={{ width: '60px' }}>
=======
    <Typography
        noWrap
        style={{ width: "60px" }}
    >
>>>>>>> main
        {status}
    </Typography>
);

export const ResourceWorkflowStatus = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
    const uuidPrefix = getUuidPrefix(state);
    return {
<<<<<<< HEAD
        ownerUuid: resource ? resource.ownerUuid : '',
=======
        ownerUuid: resource ? resource.ownerUuid : "",
>>>>>>> main
        uuidPrefix,
    };
})((props: { ownerUuid?: string; uuidPrefix: string }) => renderWorkflowStatus(props.uuidPrefix, props.ownerUuid));

export const ResourceContainerUuid = connect((state: RootState, props: { uuid: string }) => {
    const process = getProcess(props.uuid)(state.resources);
<<<<<<< HEAD
    return { uuid: process?.container?.uuid ? process?.container?.uuid : '' };
})((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

enum ColumnSelection {
    OUTPUT_UUID = 'outputUuid',
    LOG_UUID = 'logUuid',
=======
    return { uuid: process?.container?.uuid ? process?.container?.uuid : "" };
})((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

enum ColumnSelection {
    OUTPUT_UUID = "outputUuid",
    LOG_UUID = "logUuid",
>>>>>>> main
}

const renderUuidLinkWithCopyIcon = (dispatch: Dispatch, item: ProcessResource, column: string) => {
    const selectedColumnUuid = item[column];
    return (
<<<<<<< HEAD
        <Grid container alignItems='center' wrap='nowrap'>
            <Grid item>
                {selectedColumnUuid ? (
                    <Typography color='primary' style={{ width: 'auto', cursor: 'pointer' }} noWrap onClick={() => dispatch<any>(navigateTo(selectedColumnUuid))}>
                        {selectedColumnUuid}
                    </Typography>
                ) : (
                    '-'
=======
        <Grid
            container
            alignItems="center"
            wrap="nowrap"
        >
            <Grid item>
                {selectedColumnUuid ? (
                    <Typography
                        color="primary"
                        style={{ width: "auto", cursor: "pointer" }}
                        noWrap
                        onClick={() => dispatch<any>(navigateTo(selectedColumnUuid))}
                    >
                        {selectedColumnUuid}
                    </Typography>
                ) : (
                    "-"
>>>>>>> main
                )}
            </Grid>
            <Grid item>{selectedColumnUuid && renderUuidCopyIcon({ uuid: selectedColumnUuid })}</Grid>
        </Grid>
    );
};

export const ResourceOutputUuid = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<ProcessResource>(props.uuid)(state.resources);
    return resource;
})((process: ProcessResource & DispatchProp<any>) => renderUuidLinkWithCopyIcon(process.dispatch, process, ColumnSelection.OUTPUT_UUID));

export const ResourceLogUuid = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<ProcessResource>(props.uuid)(state.resources);
    return resource;
})((process: ProcessResource & DispatchProp<any>) => renderUuidLinkWithCopyIcon(process.dispatch, process, ColumnSelection.LOG_UUID));

export const ResourceParentProcess = connect((state: RootState, props: { uuid: string }) => {
    const process = getProcess(props.uuid)(state.resources);
<<<<<<< HEAD
    return { parentProcess: process?.containerRequest?.requestingContainerUuid || '' };
=======
    return { parentProcess: process?.containerRequest?.requestingContainerUuid || "" };
>>>>>>> main
})((props: { parentProcess: string }) => renderUuid({ uuid: props.parentProcess }));

export const ResourceModifiedByUserUuid = connect((state: RootState, props: { uuid: string }) => {
    const process = getProcess(props.uuid)(state.resources);
<<<<<<< HEAD
    return { userUuid: process?.containerRequest?.modifiedByUserUuid || '' };
=======
    return { userUuid: process?.containerRequest?.modifiedByUserUuid || "" };
>>>>>>> main
})((props: { userUuid: string }) => renderUuid({ uuid: props.userUuid }));

export const ResourceCreatedAtDate = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { date: resource ? resource.createdAt : '' };
=======
    return { date: resource ? resource.createdAt : "" };
>>>>>>> main
})((props: { date: string }) => renderDate(props.date));

export const ResourceLastModifiedDate = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { date: resource ? resource.modifiedAt : '' };
=======
    return { date: resource ? resource.modifiedAt : "" };
>>>>>>> main
})((props: { date: string }) => renderDate(props.date));

export const ResourceTrashDate = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<TrashableResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { date: resource ? resource.trashAt : '' };
=======
    return { date: resource ? resource.trashAt : "" };
>>>>>>> main
})((props: { date: string }) => renderDate(props.date));

export const ResourceDeleteDate = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<TrashableResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { date: resource ? resource.deleteAt : '' };
})((props: { date: string }) => renderDate(props.date));

export const renderFileSize = (fileSize?: number) => (
    <Typography noWrap style={{ minWidth: '45px' }}>
=======
    return { date: resource ? resource.deleteAt : "" };
})((props: { date: string }) => renderDate(props.date));

export const renderFileSize = (fileSize?: number) => (
    <Typography
        noWrap
        style={{ minWidth: "45px" }}
    >
>>>>>>> main
        {formatFileSize(fileSize)}
    </Typography>
);

export const ResourceFileSize = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<CollectionResource>(props.uuid)(state.resources);

    if (resource && resource.kind !== ResourceKind.COLLECTION) {
<<<<<<< HEAD
        return { fileSize: '' };
=======
        return { fileSize: "" };
>>>>>>> main
    }

    return { fileSize: resource ? resource.fileSizeTotal : 0 };
})((props: { fileSize?: number }) => renderFileSize(props.fileSize));

<<<<<<< HEAD
const renderOwner = (owner: string) => <Typography noWrap>{owner || '-'}</Typography>;

export const ResourceOwner = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
    return { owner: resource ? resource.ownerUuid : '' };
=======
const renderOwner = (owner: string) => <Typography noWrap>{owner || "-"}</Typography>;

export const ResourceOwner = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
    return { owner: resource ? resource.ownerUuid : "" };
>>>>>>> main
})((props: { owner: string }) => renderOwner(props.owner));

export const ResourceOwnerName = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
    const ownerNameState = state.ownerName;
<<<<<<< HEAD
    const ownerName = ownerNameState.find((it) => it.uuid === resource!.ownerUuid);
=======
    const ownerName = ownerNameState.find(it => it.uuid === resource!.ownerUuid);
>>>>>>> main
    return { owner: ownerName ? ownerName!.name : resource!.ownerUuid };
})((props: { owner: string }) => renderOwner(props.owner));

export const ResourceUUID = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<CollectionResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { uuid: resource ? resource.uuid : '' };
})((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

const renderVersion = (version: number) => {
    return <Typography>{version ?? '-'}</Typography>;
=======
    return { uuid: resource ? resource.uuid : "" };
})((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

const renderVersion = (version: number) => {
    return <Typography>{version ?? "-"}</Typography>;
>>>>>>> main
};

export const ResourceVersion = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<CollectionResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { version: resource ? resource.version : '' };
=======
    return { version: resource ? resource.version : "" };
>>>>>>> main
})((props: { version: number }) => renderVersion(props.version));

const renderPortableDataHash = (portableDataHash: string | null) => (
    <Typography noWrap>
        {portableDataHash ? (
            <>
                {portableDataHash}
                <CopyToClipboardSnackbar value={portableDataHash} />
            </>
        ) : (
<<<<<<< HEAD
            '-'
=======
            "-"
>>>>>>> main
        )}
    </Typography>
);

export const ResourcePortableDataHash = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<CollectionResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { portableDataHash: resource ? resource.portableDataHash : '' };
})((props: { portableDataHash: string }) => renderPortableDataHash(props.portableDataHash));

const renderFileCount = (fileCount: number) => {
    return <Typography>{fileCount ?? '-'}</Typography>;
=======
    return { portableDataHash: resource ? resource.portableDataHash : "" };
})((props: { portableDataHash: string }) => renderPortableDataHash(props.portableDataHash));

const renderFileCount = (fileCount: number) => {
    return <Typography>{fileCount ?? "-"}</Typography>;
>>>>>>> main
};

export const ResourceFileCount = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<CollectionResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { fileCount: resource ? resource.fileCount : '' };
})((props: { fileCount: number }) => renderFileCount(props.fileCount));

const userFromID = connect((state: RootState, props: { uuid: string }) => {
    let userFullname = '';
=======
    return { fileCount: resource ? resource.fileCount : "" };
})((props: { fileCount: number }) => renderFileCount(props.fileCount));

const userFromID = connect((state: RootState, props: { uuid: string }) => {
    let userFullname = "";
>>>>>>> main
    const resource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);

    if (resource) {
        userFullname = getUserFullname(resource as User) || (resource as GroupContentsResource).name;
    }

    return { uuid: props.uuid, userFullname };
});

const ownerFromResourceId = compose(
    connect((state: RootState, props: { uuid: string }) => {
        const childResource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);
<<<<<<< HEAD
        return { uuid: childResource ? (childResource as Resource).ownerUuid : '' };
=======
        return { uuid: childResource ? (childResource as Resource).ownerUuid : "" };
>>>>>>> main
    }),
    userFromID
);

const _resourceWithName = withStyles(
    {},
    { withTheme: true }
)((props: { uuid: string; userFullname: string; dispatch: Dispatch; theme: ArvadosTheme }) => {
    const { uuid, userFullname, dispatch, theme } = props;
<<<<<<< HEAD
    if (userFullname === '') {
        dispatch<any>(loadResource(uuid, false));
        return (
            <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
=======
    if (userFullname === "") {
        dispatch<any>(loadResource(uuid, false));
        return (
            <Typography
                style={{ color: theme.palette.primary.main }}
                inline
                noWrap
            >
>>>>>>> main
                {uuid}
            </Typography>
        );
    }

    return (
<<<<<<< HEAD
        <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
=======
        <Typography
            style={{ color: theme.palette.primary.main }}
            inline
            noWrap
        >
>>>>>>> main
            {userFullname} ({uuid})
        </Typography>
    );
});

export const ResourceOwnerWithName = ownerFromResourceId(_resourceWithName);

export const ResourceWithName = userFromID(_resourceWithName);

export const UserNameFromID = compose(userFromID)((props: { uuid: string; displayAsText?: string; userFullname: string; dispatch: Dispatch }) => {
    const { uuid, userFullname, dispatch } = props;

<<<<<<< HEAD
    if (userFullname === '') {
=======
    if (userFullname === "") {
>>>>>>> main
        dispatch<any>(loadResource(uuid, false));
    }
    return <span>{userFullname ? userFullname : uuid}</span>;
});

export const ResponsiblePerson = compose(
    connect((state: RootState, props: { uuid: string; parentRef: HTMLElement | null }) => {
<<<<<<< HEAD
        let responsiblePersonName: string = '';
        let responsiblePersonUUID: string = '';
        let responsiblePersonProperty: string = '';
=======
        let responsiblePersonName: string = "";
        let responsiblePersonUUID: string = "";
        let responsiblePersonProperty: string = "";
>>>>>>> main

        if (state.auth.config.clusterConfig.Collections.ManagedProperties) {
            let index = 0;
            const keys = Object.keys(state.auth.config.clusterConfig.Collections.ManagedProperties);

            while (!responsiblePersonProperty && keys[index]) {
                const key = keys[index];
<<<<<<< HEAD
                if (state.auth.config.clusterConfig.Collections.ManagedProperties[key].Function === 'original_owner') {
=======
                if (state.auth.config.clusterConfig.Collections.ManagedProperties[key].Function === "original_owner") {
>>>>>>> main
                    responsiblePersonProperty = key;
                }
                index++;
            }
        }

        let resource: Resource | undefined = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);

        while (resource && resource.kind !== ResourceKind.USER && responsiblePersonProperty) {
            responsiblePersonUUID = (resource as CollectionResource).properties[responsiblePersonProperty];
            resource = getResource<GroupContentsResource & UserResource>(responsiblePersonUUID)(state.resources);
        }

        if (resource && resource.kind === ResourceKind.USER) {
            responsiblePersonName = getUserFullname(resource as UserResource) || (resource as GroupContentsResource).name;
        }

        return { uuid: responsiblePersonUUID, responsiblePersonName, parentRef: props.parentRef };
    }),
    withStyles({}, { withTheme: true })
)((props: { uuid: string | null; responsiblePersonName: string; parentRef: HTMLElement | null; theme: ArvadosTheme }) => {
    const { uuid, responsiblePersonName, parentRef, theme } = props;

    if (!uuid && parentRef) {
<<<<<<< HEAD
        parentRef.style.display = 'none';
        return null;
    } else if (parentRef) {
        parentRef.style.display = 'block';
=======
        parentRef.style.display = "none";
        return null;
    } else if (parentRef) {
        parentRef.style.display = "block";
>>>>>>> main
    }

    if (!responsiblePersonName) {
        return (
<<<<<<< HEAD
            <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
=======
            <Typography
                style={{ color: theme.palette.primary.main }}
                inline
                noWrap
            >
>>>>>>> main
                {uuid}
            </Typography>
        );
    }

    return (
<<<<<<< HEAD
        <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
=======
        <Typography
            style={{ color: theme.palette.primary.main }}
            inline
            noWrap
        >
>>>>>>> main
            {responsiblePersonName} ({uuid})
        </Typography>
    );
});

const renderType = (type: string, subtype: string) => <Typography noWrap>{resourceLabel(type, subtype)}</Typography>;

export const ResourceType = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
<<<<<<< HEAD
    return { type: resource ? resource.kind : '', subtype: resource && resource.kind === ResourceKind.GROUP ? resource.groupClass : '' };
=======
    return { type: resource ? resource.kind : "", subtype: resource && resource.kind === ResourceKind.GROUP ? resource.groupClass : "" };
>>>>>>> main
})((props: { type: string; subtype: string }) => renderType(props.type, props.subtype));

export const ResourceStatus = connect((state: RootState, props: { uuid: string }) => {
    return { resource: getResource<GroupContentsResource>(props.uuid)(state.resources) };
})((props: { resource: GroupContentsResource }) =>
<<<<<<< HEAD
    props.resource && props.resource.kind === ResourceKind.COLLECTION ? <CollectionStatus uuid={props.resource.uuid} /> : <ProcessStatus uuid={props.resource.uuid} />
=======
    props.resource && props.resource.kind === ResourceKind.COLLECTION ? (
        <CollectionStatus uuid={props.resource.uuid} />
    ) : (
        <ProcessStatus uuid={props.resource.uuid} />
    )
>>>>>>> main
);

export const CollectionStatus = connect((state: RootState, props: { uuid: string }) => {
    return { collection: getResource<CollectionResource>(props.uuid)(state.resources) };
})((props: { collection: CollectionResource }) =>
<<<<<<< HEAD
    props.collection.uuid !== props.collection.currentVersionUuid ? <Typography>version {props.collection.version}</Typography> : <Typography>head version</Typography>
=======
    props.collection.uuid !== props.collection.currentVersionUuid ? (
        <Typography>version {props.collection.version}</Typography>
    ) : (
        <Typography>head version</Typography>
    )
>>>>>>> main
);

export const CollectionName = connect((state: RootState, props: { uuid: string; className?: string }) => {
    return {
        collection: getResource<CollectionResource>(props.uuid)(state.resources),
        uuid: props.uuid,
        className: props.className,
    };
})((props: { collection: CollectionResource; uuid: string; className?: string }) => (
    <Typography className={props.className}>{props.collection?.name || props.uuid}</Typography>
));

export const ProcessStatus = compose(
    connect((state: RootState, props: { uuid: string }) => {
        return { process: getProcess(props.uuid)(state.resources) };
    }),
    withStyles({}, { withTheme: true })
)((props: { process?: Process; theme: ArvadosTheme }) =>
    props.process ? (
        <Chip
            label={getProcessStatus(props.process)}
            style={{
                height: props.theme.spacing.unit * 3,
                width: props.theme.spacing.unit * 12,
                ...getProcessStatusStyles(getProcessStatus(props.process), props.theme),
<<<<<<< HEAD
                fontSize: '0.875rem',
=======
                fontSize: "0.875rem",
>>>>>>> main
                borderRadius: props.theme.spacing.unit * 0.625,
            }}
        />
    ) : (
        <Typography>-</Typography>
    )
);

export const ProcessStartDate = connect((state: RootState, props: { uuid: string }) => {
    const process = getProcess(props.uuid)(state.resources);
<<<<<<< HEAD
    return { date: process && process.container ? process.container.startedAt : '' };
})((props: { date: string }) => renderDate(props.date));

export const renderRunTime = (time: number) => (
    <Typography noWrap style={{ minWidth: '45px' }}>
=======
    return { date: process && process.container ? process.container.startedAt : "" };
})((props: { date: string }) => renderDate(props.date));

export const renderRunTime = (time: number) => (
    <Typography
        noWrap
        style={{ minWidth: "45px" }}
    >
>>>>>>> main
        {formatTime(time, true)}
    </Typography>
);

interface ContainerRunTimeProps {
    process: Process;
}

interface ContainerRunTimeState {
    runtime: number;
}

export const ContainerRunTime = connect((state: RootState, props: { uuid: string }) => {
    return { process: getProcess(props.uuid)(state.resources) };
})(
    class extends React.Component<ContainerRunTimeProps, ContainerRunTimeState> {
        private timer: any;

        constructor(props: ContainerRunTimeProps) {
            super(props);
            this.state = { runtime: this.getRuntime() };
        }

        getRuntime() {
            return this.props.process ? getProcessRuntime(this.props.process) : 0;
        }

        updateRuntime() {
            this.setState({ runtime: this.getRuntime() });
        }

        componentDidMount() {
            this.timer = setInterval(this.updateRuntime.bind(this), 5000);
        }

        componentWillUnmount() {
            clearInterval(this.timer);
        }

        render() {
            return this.props.process ? renderRunTime(this.state.runtime) : <Typography>-</Typography>;
        }
    }
);
