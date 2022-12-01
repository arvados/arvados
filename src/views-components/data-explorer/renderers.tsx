// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    Grid,
    Typography,
    withStyles,
    Tooltip,
    IconButton,
    Checkbox,
    Chip
} from '@material-ui/core';
import { FavoriteStar, PublicFavoriteStar } from '../favorite-star/favorite-star';
import { Resource, ResourceKind, TrashableResource } from 'models/resource';
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
} from 'components/icon/icon';
import { formatDate, formatFileSize, formatTime } from 'common/formatters';
import { resourceLabel } from 'common/labels';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { getResource, filterResources } from 'store/resources/resources';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { getProcess, Process, getProcessStatus, getProcessStatusColor, getProcessRuntime } from 'store/processes/process';
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

const renderName = (dispatch: Dispatch, item: GroupContentsResource) => {

    const navFunc = ("groupClass" in item && item.groupClass === GroupClass.ROLE ? navigateToGroupDetails : navigateTo);
    return <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary" style={{ width: 'auto', cursor: 'pointer' }} onClick={() => dispatch<any>(navFunc(item.uuid))}>
                {item.kind === ResourceKind.PROJECT || item.kind === ResourceKind.COLLECTION
                    ? <IllegalNamingWarning name={item.name} />
                    : null}
                {item.name}
            </Typography>
        </Grid>
        <Grid item>
            <Typography variant="caption">
                <FavoriteStar resourceUuid={item.uuid} />
                <PublicFavoriteStar resourceUuid={item.uuid} />
                {
                    item.kind === ResourceKind.PROJECT && <FrozenProject item={item} />
                }
            </Typography>
        </Grid>
    </Grid>;
};

const FrozenProject = (props: {item: ProjectResource}) => {
    const [fullUsername, setFullusername] = React.useState<any>(null);
    const getFullName = React.useCallback(() => {
        if (props.item.frozenByUuid) {
            setFullusername(<UserNameFromID uuid={props.item.frozenByUuid} />);
        }
    }, [props.item, setFullusername])

    if (props.item.frozenByUuid) {

        return <Tooltip onOpen={getFullName} enterDelay={500} title={<span>Project was frozen by {fullUsername}</span>}>
            <FreezeIcon style={{ fontSize: "inherit" }}/>
        </Tooltip>;
    } else {
        return null;
    }
}

export const ResourceName = connect(
    (state: RootState, props: { uuid: string }) => {
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
    return <Typography noWrap style={{ minWidth: '100px' }}>{formatDate(date)}</Typography>;
};

const renderWorkflowName = (item: WorkflowResource) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary" style={{ width: '100px' }}>
                {item.name}
            </Typography>
        </Grid>
    </Grid>;

export const ResourceWorkflowName = connect(
    (state: RootState, props: { uuid: string }) => {
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
            {!isPublic && uuid &&
                <Tooltip title="Share">
                    <IconButton onClick={() => dispatch<any>(openSharingDialog(uuid))}>
                        <ShareIcon />
                    </IconButton>
                </Tooltip>
            }
        </div>
    );
};

export const ResourceShare = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        const uuidPrefix = getUuidPrefix(state);
        return {
            uuid: resource ? resource.uuid : '',
            ownerUuid: resource ? resource.ownerUuid : '',
            uuidPrefix
        };
    })((props: { ownerUuid?: string, uuidPrefix: string, uuid?: string } & DispatchProp<any>) =>
        resourceShare(props.dispatch, props.uuidPrefix, props.ownerUuid, props.uuid));

// User Resources
const renderFirstName = (item: { firstName: string }) => {
    return <Typography noWrap>{item.firstName}</Typography>;
};

export const ResourceFirstName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { firstName: '' };
    })(renderFirstName);

const renderLastName = (item: { lastName: string }) =>
    <Typography noWrap>{item.lastName}</Typography>;

export const ResourceLastName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { lastName: '' };
    })(renderLastName);

const renderFullName = (dispatch: Dispatch, item: { uuid: string, firstName: string, lastName: string }, link?: boolean) => {
    const displayName = (item.firstName + " " + item.lastName).trim() || item.uuid;
    return link ? <Typography noWrap
        color="primary"
        style={{ 'cursor': 'pointer' }}
        onClick={() => dispatch<any>(navigateToUserProfile(item.uuid))}>
        {displayName}
    </Typography> :
        <Typography noWrap>{displayName}</Typography>;
}

export const UserResourceFullName = connect(
    (state: RootState, props: { uuid: string, link?: boolean }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return { item: resource || { uuid: '', firstName: '', lastName: '' }, link: props.link };
    })((props: { item: { uuid: string, firstName: string, lastName: string }, link?: boolean } & DispatchProp<any>) => renderFullName(props.dispatch, props.item, props.link));

const renderUuid = (item: { uuid: string }) =>
    <Typography data-cy="uuid" noWrap>
        {item.uuid}
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || '-' }
    </Typography>;

export const ResourceUuid = connect((state: RootState, props: { uuid: string }) => (
    getResource<UserResource>(props.uuid)(state.resources) || { uuid: '' }
))(renderUuid);

const renderEmail = (item: { email: string }) =>
    <Typography noWrap>{item.email}</Typography>;

export const ResourceEmail = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { email: '' };
    })(renderEmail);

enum UserAccountStatus {
    ACTIVE = 'Active',
    INACTIVE = 'Inactive',
    SETUP = 'Setup',
    UNKNOWN = ''
}

const renderAccountStatus = (props: { status: UserAccountStatus }) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={8} data-cy="account-status">
        <Grid item>
            {(() => {
                switch (props.status) {
                    case UserAccountStatus.ACTIVE:
                        return <ActiveIcon style={{ color: '#4caf50', verticalAlign: "middle" }} />;
                    case UserAccountStatus.SETUP:
                        return <SetupIcon style={{ color: '#2196f3', verticalAlign: "middle" }} />;
                    case UserAccountStatus.INACTIVE:
                        return <InactiveIcon style={{ color: '#9e9e9e', verticalAlign: "middle" }} />;
                    default:
                        return <></>;
                }
            })()}
        </Grid>
        <Grid item>
            <Typography noWrap>
                {props.status}
            </Typography>
        </Grid>
    </Grid>;

const getUserAccountStatus = (state: RootState, props: { uuid: string }) => {
    const user = getResource<UserResource>(props.uuid)(state.resources);
    // Get membership links for all users group
    const allUsersGroupUuid = getBuiltinGroupUuid(state.auth.localCluster, BuiltinGroups.ALL);
    const permissions = filterResources((resource: LinkResource) =>
        resource.kind === ResourceKind.LINK &&
        resource.linkClass === LinkClass.PERMISSION &&
        resource.headUuid === allUsersGroupUuid &&
        resource.tailUuid === props.uuid
    )(state.resources);

    if (user) {
        return user.isActive ? { status: UserAccountStatus.ACTIVE } : permissions.length > 0 ? { status: UserAccountStatus.SETUP } : { status: UserAccountStatus.INACTIVE };
    } else {
        return { status: UserAccountStatus.UNKNOWN };
    }
}

export const ResourceLinkTailAccountStatus = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        return link && link.tailKind === ResourceKind.USER ? getUserAccountStatus(state, { uuid: link.tailUuid }) : { status: UserAccountStatus.UNKNOWN };
    })(renderAccountStatus);

export const UserResourceAccountStatus = connect(getUserAccountStatus)(renderAccountStatus);

const renderIsHidden = (props: {
    memberLinkUuid: string,
    permissionLinkUuid: string,
    visible: boolean,
    canManage: boolean,
    setMemberIsHidden: (memberLinkUuid: string, permissionLinkUuid: string, hide: boolean) => void
}) => {
    if (props.memberLinkUuid) {
        return <Checkbox
            data-cy="user-visible-checkbox"
            color="primary"
            checked={props.visible}
            disabled={!props.canManage}
            onClick={(e) => {
                e.stopPropagation();
                props.setMemberIsHidden(props.memberLinkUuid, props.permissionLinkUuid, !props.visible);
            }} />;
    } else {
        return <Typography />;
    }
}

export const ResourceLinkTailIsVisible = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const member = getResource<Resource>(link?.tailUuid || '')(state.resources);
        const group = getResource<GroupResource>(link?.headUuid || '')(state.resources);
        const permissions = filterResources((resource: LinkResource) => {
            return resource.linkClass === LinkClass.PERMISSION
                && resource.headUuid === link?.tailUuid
                && resource.tailUuid === group?.uuid
                && resource.name === PermissionLevel.CAN_READ;
        })(state.resources);

        const permissionLinkUuid = permissions.length > 0 ? permissions[0].uuid : '';
        const isVisible = link && group && permissions.length > 0;
        // Consider whether the current user canManage this resurce in addition when it's possible
        const isBuiltin = isBuiltinGroup(link?.headUuid || '');

        return member?.kind === ResourceKind.USER
            ? { memberLinkUuid: link?.uuid, permissionLinkUuid, visible: isVisible, canManage: !isBuiltin }
            : { memberLinkUuid: '', permissionLinkUuid: '', visible: false, canManage: false };
    }, { setMemberIsHidden }
)(renderIsHidden);

const renderIsAdmin = (props: { uuid: string, isAdmin: boolean, toggleIsAdmin: (uuid: string) => void }) =>
    <Checkbox
        color="primary"
        checked={props.isAdmin}
        onClick={(e) => {
            e.stopPropagation();
            props.toggleIsAdmin(props.uuid);
        }} />;

export const ResourceIsAdmin = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { isAdmin: false };
    }, { toggleIsAdmin }
)(renderIsAdmin);

const renderUsername = (item: { username: string, uuid: string }) =>
    <Typography noWrap>{item.username || item.uuid}</Typography>;

export const ResourceUsername = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { username: '', uuid: props.uuid };
    })(renderUsername);

// Virtual machine resource

const renderHostname = (item: { hostname: string }) =>
    <Typography noWrap>{item.hostname}</Typography>;

export const VirtualMachineHostname = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<VirtualMachinesResource>(props.uuid)(state.resources);
        return resource || { hostname: '' };
    })(renderHostname);

const renderVirtualMachineLogin = (login: { user: string }) =>
    <Typography noWrap>{login.user}</Typography>

export const VirtualMachineLogin = connect(
    (state: RootState, props: { linkUuid: string }) => {
        const permission = getResource<LinkResource>(props.linkUuid)(state.resources);
        const user = getResource<UserResource>(permission?.tailUuid || '')(state.resources);

        return { user: user?.username || permission?.tailUuid || '' };
    })(renderVirtualMachineLogin);

// Common methods
const renderCommonData = (data: string) =>
    <Typography noWrap>{data}</Typography>;

const renderCommonDate = (date: string) =>
    <Typography noWrap>{formatDate(date)}</Typography>;

export const CommonUuid = withResourceData('uuid', renderCommonData);

// Api Client Authorizations
export const TokenApiClientId = withResourceData('apiClientId', renderCommonData);

export const TokenApiToken = withResourceData('apiToken', renderCommonData);

export const TokenCreatedByIpAddress = withResourceData('createdByIpAddress', renderCommonDate);

export const TokenDefaultOwnerUuid = withResourceData('defaultOwnerUuid', renderCommonData);

export const TokenExpiresAt = withResourceData('expiresAt', renderCommonDate);

export const TokenLastUsedAt = withResourceData('lastUsedAt', renderCommonDate);

export const TokenLastUsedByIpAddress = withResourceData('lastUsedByIpAddress', renderCommonData);

export const TokenScopes = withResourceData('scopes', renderCommonData);

export const TokenUserId = withResourceData('userId', renderCommonData);

const clusterColors = [
    ['#f44336', '#fff'],
    ['#2196f3', '#fff'],
    ['#009688', '#fff'],
    ['#cddc39', '#fff'],
    ['#ff9800', '#fff']
];

export const ResourceCluster = (props: { uuid: string }) => {
    const CLUSTER_ID_LENGTH = 5;
    const pos = props.uuid.length > CLUSTER_ID_LENGTH ? props.uuid.indexOf('-') : 5;
    const clusterId = pos >= CLUSTER_ID_LENGTH ? props.uuid.substring(0, pos) : '';
    const ci = pos >= CLUSTER_ID_LENGTH ? (((((
        (props.uuid.charCodeAt(0) * props.uuid.charCodeAt(1))
        + props.uuid.charCodeAt(2))
        * props.uuid.charCodeAt(3))
        + props.uuid.charCodeAt(4))) % clusterColors.length) : 0;
    return <span style={{
        backgroundColor: clusterColors[ci][0],
        color: clusterColors[ci][1],
        padding: "2px 7px",
        borderRadius: 3
    }}>{clusterId}</span>;
};

// Links Resources
const renderLinkName = (item: { name: string }) =>
    <Typography noWrap>{item.name || '(none)'}</Typography>;

export const ResourceLinkName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return resource || { name: '' };
    })(renderLinkName);

const renderLinkClass = (item: { linkClass: string }) =>
    <Typography noWrap>{item.linkClass}</Typography>;

export const ResourceLinkClass = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return resource || { linkClass: '' };
    })(renderLinkClass);

const getResourceDisplayName = (resource: Resource): string => {
    if ((resource as UserResource).kind === ResourceKind.USER
        && typeof (resource as UserResource).firstName !== 'undefined') {
        // We can be sure the resource is UserResource
        return getUserDisplayName(resource as UserResource);
    } else {
        return (resource as GroupContentsResource).name;
    }
}

const renderResourceLink = (dispatch: Dispatch, item: Resource) => {
    var displayName = getResourceDisplayName(item);

    return <Typography noWrap color="primary" style={{ 'cursor': 'pointer' }} onClick={() => dispatch<any>(navigateTo(item.uuid))}>
        {resourceLabel(item.kind, item && item.kind === ResourceKind.GROUP ? (item as GroupResource).groupClass || '' : '')}: {displayName || item.uuid}
    </Typography>;
};

export const ResourceLinkTail = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        const tailResource = getResource<Resource>(resource?.tailUuid || '')(state.resources);

        return {
            item: tailResource || { uuid: resource?.tailUuid || '', kind: resource?.tailKind || ResourceKind.NONE }
        };
    })((props: { item: Resource } & DispatchProp<any>) =>
        renderResourceLink(props.dispatch, props.item));

export const ResourceLinkHead = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        const headResource = getResource<Resource>(resource?.headUuid || '')(state.resources);

        return {
            item: headResource || { uuid: resource?.headUuid || '', kind: resource?.headKind || ResourceKind.NONE }
        };
    })((props: { item: Resource } & DispatchProp<any>) =>
        renderResourceLink(props.dispatch, props.item));

export const ResourceLinkUuid = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return resource || { uuid: '' };
    })(renderUuid);

export const ResourceLinkHeadUuid = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const headResource = getResource<Resource>(link?.headUuid || '')(state.resources);

        return headResource || { uuid: '' };
    })(renderUuid);

export const ResourceLinkTailUuid = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const tailResource = getResource<Resource>(link?.tailUuid || '')(state.resources);

        return tailResource || { uuid: '' };
    })(renderUuid);

const renderLinkDelete = (dispatch: Dispatch, item: LinkResource, canManage: boolean) => {
    if (item.uuid) {
        return canManage ?
            <Typography noWrap>
                <IconButton data-cy="resource-delete-button" onClick={() => dispatch<any>(openRemoveGroupMemberDialog(item.uuid))}>
                    <RemoveIcon />
                </IconButton>
            </Typography> :
            <Typography noWrap>
                <IconButton disabled data-cy="resource-delete-button">
                    <RemoveIcon />
                </IconButton>
            </Typography>;
    } else {
        return <Typography noWrap></Typography>;
    }
}

export const ResourceLinkDelete = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const isBuiltin = isBuiltinGroup(link?.headUuid || '') || isBuiltinGroup(link?.tailUuid || '');

        return {
            item: link || { uuid: '', kind: ResourceKind.NONE },
            canManage: link && getResourceLinkCanManage(state, link) && !isBuiltin,
        };
    })((props: { item: LinkResource, canManage: boolean } & DispatchProp<any>) =>
        renderLinkDelete(props.dispatch, props.item, props.canManage));

export const ResourceLinkTailEmail = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const resource = getResource<UserResource>(link?.tailUuid || '')(state.resources);

        return resource || { email: '' };
    })(renderEmail);

export const ResourceLinkTailUsername = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const resource = getResource<UserResource>(link?.tailUuid || '')(state.resources);

        return resource || { username: '' };
    })(renderUsername);

const renderPermissionLevel = (dispatch: Dispatch, link: LinkResource, canManage: boolean) => {
    return <Typography noWrap>
        {formatPermissionLevel(link.name as PermissionLevel)}
        {canManage ?
            <IconButton data-cy="edit-permission-button" onClick={(event) => dispatch<any>(openPermissionEditContextMenu(event, link))}>
                <RenameIcon />
            </IconButton> :
            ''
        }
    </Typography>;
}

export const ResourceLinkHeadPermissionLevel = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const isBuiltin = isBuiltinGroup(link?.headUuid || '') || isBuiltinGroup(link?.tailUuid || '');

        return {
            link: link || { uuid: '', name: '', kind: ResourceKind.NONE },
            canManage: link && getResourceLinkCanManage(state, link) && !isBuiltin,
        };
    })((props: { link: LinkResource, canManage: boolean } & DispatchProp<any>) =>
        renderPermissionLevel(props.dispatch, props.link, props.canManage));

export const ResourceLinkTailPermissionLevel = connect(
    (state: RootState, props: { uuid: string }) => {
        const link = getResource<LinkResource>(props.uuid)(state.resources);
        const isBuiltin = isBuiltinGroup(link?.headUuid || '') || isBuiltinGroup(link?.tailUuid || '');

        return {
            link: link || { uuid: '', name: '', kind: ResourceKind.NONE },
            canManage: link && getResourceLinkCanManage(state, link) && !isBuiltin,
        };
    })((props: { link: LinkResource, canManage: boolean } & DispatchProp<any>) =>
        renderPermissionLevel(props.dispatch, props.link, props.canManage));

const getResourceLinkCanManage = (state: RootState, link: LinkResource) => {
    const headResource = getResource<Resource>(link.headUuid)(state.resources);
    // const tailResource = getResource<Resource>(link.tailUuid)(state.resources);
    const userUuid = getUserUuid(state);

    if (headResource && headResource.kind === ResourceKind.GROUP) {
        return userUuid ? (headResource as GroupResource).writableBy?.includes(userUuid) : false;
    } else {
        // true for now
        return true;
    }
}

// Process Resources
const resourceRunProcess = (dispatch: Dispatch, uuid: string) => {
    return (
        <div>
            {uuid &&
                <Tooltip title="Run process">
                    <IconButton onClick={() => dispatch<any>(openRunProcess(uuid))}>
                        <ProcessIcon />
                    </IconButton>
                </Tooltip>}
        </div>
    );
};

export const ResourceRunProcess = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        return {
            uuid: resource ? resource.uuid : ''
        };
    })((props: { uuid: string } & DispatchProp<any>) =>
        resourceRunProcess(props.dispatch, props.uuid));

const renderWorkflowStatus = (uuidPrefix: string, ownerUuid?: string) => {
    if (ownerUuid === getPublicUuid(uuidPrefix)) {
        return renderStatus(WorkflowStatus.PUBLIC);
    } else {
        return renderStatus(WorkflowStatus.PRIVATE);
    }
};

const renderStatus = (status: string) =>
    <Typography noWrap style={{ width: '60px' }}>{status}</Typography>;

export const ResourceWorkflowStatus = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        const uuidPrefix = getUuidPrefix(state);
        return {
            ownerUuid: resource ? resource.ownerUuid : '',
            uuidPrefix
        };
    })((props: { ownerUuid?: string, uuidPrefix: string }) => renderWorkflowStatus(props.uuidPrefix, props.ownerUuid));

const renderProcessState = (processState: string) => <Typography>{processState || '-'}</Typography>

export const ResourceProcessState = connect(
    (state: RootState, props: { uuid: string }) => {
        const process = getProcess(props.uuid)(state.resources)
        // console.log('PROCESS>>>', process)
        return { state: process?.container?.state ? process?.container?.state : '' };
    })((props: { state: string }) => renderProcessState(props.state));

export const ResourceProcessUuid = connect(
    (state: RootState, props: { uuid: string }) => {
        const process = getProcess(props.uuid)(state.resources)
        return { uuid: process?.container?.uuid ? process?.container?.uuid : '' };
    })((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

export const ResourceParentProcess = connect(
    (state: RootState, props: { uuid: string }) => {
        const process = getProcess(props.uuid)(state.resources)
        const parentProcessUuid = process?.containerRequest?.requestingContainerUuid
        return { parentProcess: parentProcessUuid || '' };
    })((props: { parentProcess: string }) => renderUuid({uuid: props.parentProcess}));

export const ResourceCreatedAtDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { date: resource ? resource.createdAt : '' };
    })((props: { date: string }) => renderDate(props.date));
    
export const ResourceLastModifiedDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { date: resource ? resource.modifiedAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const ResourceTrashDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<TrashableResource>(props.uuid)(state.resources);
        return { date: resource ? resource.trashAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const ResourceDeleteDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<TrashableResource>(props.uuid)(state.resources);
        return { date: resource ? resource.deleteAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const renderFileSize = (fileSize?: number) =>
    <Typography noWrap style={{ minWidth: '45px' }}>
        {formatFileSize(fileSize)}
    </Typography>;

export const ResourceFileSize = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<CollectionResource>(props.uuid)(state.resources);

        if (resource && resource.kind !== ResourceKind.COLLECTION) {
            return { fileSize: '' };
        }

        return { fileSize: resource ? resource.fileSizeTotal : 0 };
    })((props: { fileSize?: number }) => renderFileSize(props.fileSize));

const renderOwner = (owner: string) =>
    <Typography noWrap>
        {owner || '-'}
    </Typography>;

export const ResourceOwner = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { owner: resource ? resource.ownerUuid : '' };
    })((props: { owner: string }) => renderOwner(props.owner));

export const ResourceOwnerName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        const ownerNameState = state.ownerName;
        const ownerName = ownerNameState.find(it => it.uuid === resource!.ownerUuid);
        return { owner: ownerName ? ownerName!.name : resource!.ownerUuid };
    })((props: { owner: string }) => renderOwner(props.owner));

export const ResourceUUID = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<CollectionResource>(props.uuid)(state.resources);
        return { uuid: resource ? resource.uuid : '' };
    })((props: { uuid: string }) => renderUuid({uuid: props.uuid}));
    
const renderPortableDataHash = (portableDataHash:string | null) => 
    <Typography noWrap>
        {portableDataHash ? <>{portableDataHash}
        <CopyToClipboardSnackbar value={portableDataHash} /></> : '-' }
    </Typography>
    
export const ResourcePortableDataHash = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<CollectionResource>(props.uuid)(state.resources);
        return { portableDataHash: resource ? resource.portableDataHash : '' };    
    })((props: { portableDataHash: string }) => renderPortableDataHash(props.portableDataHash));

const renderVersion = (version: number) =>{
    return <Typography>{version ?? '-'}</Typography>
}

export const ResourceVersion = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<CollectionResource>(props.uuid)(state.resources);
        return { version: resource ? resource.version: '' };
    })((props: { version: number }) => renderVersion(props.version));

const renderDescription = (description: string)=>{
    const truncatedDescription = description ? description.slice(0, 18) + '...' : '-'
    return <Typography title={description}>{truncatedDescription}</Typography>;
}

const renderFileCount = (fileCount: number) =>{
    return <Typography>{fileCount ?? '-'}</Typography>
}

export const ResourceFileCount = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<CollectionResource>(props.uuid)(state.resources);
        return { fileCount: resource ? resource.fileCount: '' };
    })((props: { fileCount: number }) => renderFileCount(props.fileCount));

export const ResourceDescription = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        //testing---------------
        // const containerRequestDescription = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
        // if (resource && !resource.description && resource.kind === ResourceKind.PROCESS) resource.description = containerRequestDescription
        //testing---------------
        return { description: resource ? resource.description : '' };
    })((props: { description: string }) => renderDescription(props.description));

const userFromID =
    connect(
        (state: RootState, props: { uuid: string }) => {
            let userFullname = '';
            const resource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);

            if (resource) {
                userFullname = getUserFullname(resource as User) || (resource as GroupContentsResource).name;
            }

            return { uuid: props.uuid, userFullname };
        });

const ownerFromResourceId =
    compose(
        connect((state: RootState, props: { uuid: string }) => {
            const childResource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);
            return { uuid: childResource ? (childResource as Resource).ownerUuid : '' };
        }),
        userFromID
    );

const _resourceWithName =
    withStyles({}, { withTheme: true })
        ((props: { uuid: string, userFullname: string, dispatch: Dispatch, theme: ArvadosTheme }) => {
            const { uuid, userFullname, dispatch, theme } = props;

            if (userFullname === '') {
                dispatch<any>(loadResource(uuid, false));
                return <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
                    {uuid}
                </Typography>;
            }

            return <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
                {userFullname} ({uuid})
            </Typography>;
        });

export const ResourceOwnerWithName = ownerFromResourceId(_resourceWithName);

export const ResourceWithName = userFromID(_resourceWithName);

export const UserNameFromID =
    compose(userFromID)(
        (props: { uuid: string, displayAsText?: string, userFullname: string, dispatch: Dispatch }) => {
            const { uuid, userFullname, dispatch } = props;

            if (userFullname === '') {
                dispatch<any>(loadResource(uuid, false));
            }
            return <span>
                {userFullname ? userFullname : uuid}
            </span>;
        });

export const ResponsiblePerson =
    compose(
        connect(
            (state: RootState, props: { uuid: string, parentRef: HTMLElement | null }) => {
                let responsiblePersonName: string = '';
                let responsiblePersonUUID: string = '';
                let responsiblePersonProperty: string = '';

                if (state.auth.config.clusterConfig.Collections.ManagedProperties) {
                    let index = 0;
                    const keys = Object.keys(state.auth.config.clusterConfig.Collections.ManagedProperties);

                    while (!responsiblePersonProperty && keys[index]) {
                        const key = keys[index];
                        if (state.auth.config.clusterConfig.Collections.ManagedProperties[key].Function === 'original_owner') {
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
        withStyles({}, { withTheme: true }))
        ((props: { uuid: string | null, responsiblePersonName: string, parentRef: HTMLElement | null, theme: ArvadosTheme }) => {
            const { uuid, responsiblePersonName, parentRef, theme } = props;

            if (!uuid && parentRef) {
                parentRef.style.display = 'none';
                return null;
            } else if (parentRef) {
                parentRef.style.display = 'block';
            }

            if (!responsiblePersonName) {
                return <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
                    {uuid}
                </Typography>;
            }

            return <Typography style={{ color: theme.palette.primary.main }} inline noWrap>
                {responsiblePersonName} ({uuid})
            </Typography>;
        });

const renderType = (type: string, subtype: string) =>
    <Typography noWrap>
        {resourceLabel(type, subtype)}
    </Typography>;

export const ResourceType = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { type: resource ? resource.kind : '', subtype: resource && resource.kind === ResourceKind.GROUP ? resource.groupClass : '' };
    })((props: { type: string, subtype: string }) => renderType(props.type, props.subtype));

export const ResourceStatus = connect((state: RootState, props: { uuid: string }) => {
    return { resource: getResource<GroupContentsResource>(props.uuid)(state.resources) };
})((props: { resource: GroupContentsResource }) =>
    (props.resource && props.resource.kind === ResourceKind.COLLECTION)
        ? <CollectionStatus uuid={props.resource.uuid} />
        : <ProcessStatus uuid={props.resource.uuid} />
);

export const CollectionStatus = connect((state: RootState, props: { uuid: string }) => {
    return { collection: getResource<CollectionResource>(props.uuid)(state.resources) };
})((props: { collection: CollectionResource }) =>
    (props.collection.uuid !== props.collection.currentVersionUuid)
        ? <Typography>version {props.collection.version}</Typography>
        : <Typography>head version</Typography>
);

export const CollectionName = connect((state: RootState, props: { uuid: string, className?: string }) => {
    return {
                collection: getResource<CollectionResource>(props.uuid)(state.resources),
                uuid: props.uuid,
                className: props.className,
            };
})((props: { collection: CollectionResource, uuid: string, className?: string }) =>
        <Typography className={props.className}>{props.collection?.name || props.uuid}</Typography>
);

export const ProcessStatus = compose(
    connect((state: RootState, props: { uuid: string }) => {
        return { process: getProcess(props.uuid)(state.resources) };
    }),
    withStyles({}, { withTheme: true }))
    ((props: { process?: Process, theme: ArvadosTheme }) =>
        props.process
            ? <Chip label={getProcessStatus(props.process)}
                style={{
                    height: props.theme.spacing.unit * 3,
                    width: props.theme.spacing.unit * 12,
                    backgroundColor: getProcessStatusColor(
                        getProcessStatus(props.process), props.theme),
                    color: props.theme.palette.common.white,
                    fontSize: '0.875rem',
                    borderRadius: props.theme.spacing.unit * 0.625,
                }}
            />
            : <Typography>-</Typography>
    );

export const ProcessStartDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const process = getProcess(props.uuid)(state.resources);
        return { date: (process && process.container) ? process.container.startedAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const renderRunTime = (time: number) =>
    <Typography noWrap style={{ minWidth: '45px' }}>
        {formatTime(time, true)}
    </Typography>;

interface ContainerRunTimeProps {
    process: Process;
}

interface ContainerRunTimeState {
    runtime: number;
}

export const ContainerRunTime = connect((state: RootState, props: { uuid: string }) => {
    return { process: getProcess(props.uuid)(state.resources) };
})(class extends React.Component<ContainerRunTimeProps, ContainerRunTimeState> {
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
        return renderRunTime(this.state.runtime);
    }
});
