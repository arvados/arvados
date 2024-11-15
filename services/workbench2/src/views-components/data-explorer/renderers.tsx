// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Grid, Typography, Tooltip, IconButton, Checkbox, Chip } from "@mui/material";
import withStyles from '@mui/styles/withStyles';
import { FavoriteStar, PublicFavoriteStar } from "../favorite-star/favorite-star";
import { Resource, ResourceKind, TrashableResource } from "models/resource";
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
    ErrorIcon,
    RestoreFromTrashIcon,
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
import { LinkClass, LinkResource } from "models/link";
import { navigateTo, navigateToGroupDetails, navigateToUserProfile } from "store/navigation/navigation-action";
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
import { ServiceRepository } from "services/services";
import { loadUsersPanel } from "store/users/users-actions";
import { InlinePulser } from "components/loading/inline-pulser";
import { ProcessTypeFilter } from "store/resource-type-filters/resource-type-filters";
import { CustomTheme } from "common/custom-theme";
import { dispatchAction } from "common/dispatch-action";
import { getProperty } from "store/properties/properties";
import { ClusterBadge } from "store/auth/cluster-badges";
import { PermissionResource } from 'models/permission';
import { ContainerRequestResource } from 'models/container-request';
import { toggleTrashed } from "store/trash/trash-actions";

// A generic wrapper for components that need to dispatch actions
const dispatchWrapper = (component: React.ComponentType<any>) => connect()(component);

export const toggleIsAdmin = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const isAdmin = data!.isAdmin;
        const newActivity = await services.userService.update(uuid, { isAdmin: !isAdmin });
        dispatch<any>(loadUsersPanel());
        return newActivity;
    };

export const RenderName = dispatchWrapper((props: { resource: GroupContentsResource, dispatch: Dispatch }) => {
    const { resource, dispatch } = props;
    const navFunc = "groupClass" in resource && resource.groupClass === GroupClass.ROLE ? navigateToGroupDetails : navigateTo;
    return (
        <Grid
            container
            alignItems="center"
            wrap="nowrap"
            spacing={2}
        >
            <Grid item style={{color: CustomTheme.palette.grey['600'] }}>{renderIcon(resource)}</Grid>
            <Grid item>
                <Typography
                    color="primary"
                    style={{ width: "auto", cursor: "pointer" }}
                    onClick={(ev) => {
                        ev.stopPropagation()
                        dispatch<any>(navFunc(resource.uuid))
                    }}
                >
                    {resource.kind === ResourceKind.PROJECT || resource.kind === ResourceKind.COLLECTION ? <IllegalNamingWarning name={resource.name} /> : null}
                    {resource.name}
                </Typography>
            </Grid>
            <Grid item>
                <Typography variant="caption">
                    <FavoriteStar resourceUuid={resource.uuid} />
                    <PublicFavoriteStar resourceUuid={resource.uuid} />
                    {resource.kind === ResourceKind.PROJECT && <FrozenProject item={resource} />}
                </Typography>
            </Grid>
        </Grid>
    );
});

export const FrozenProject = (props: { item: ProjectResource }) => {
    const [fullUsername, setFullusername] = React.useState<any>(null);
    const getFullName = React.useCallback(() => {
        if (props.item.frozenByUuid) {
            setFullusername(<UserNameFromID uuid={props.item.frozenByUuid} />);
        }
    }, [props.item, setFullusername]);

    if (props.item.frozenByUuid) {
        return (
            <Tooltip
                onOpen={getFullName}
                enterDelay={500}
                title={<span>Project was frozen by {fullUsername}</span>}
            >
                <FreezeIcon style={{ fontSize: "inherit" }} />
            </Tooltip>
        );
    } else {
        return null;
    }
};

// export const ResourceName = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
//     return resource;
// })((resource: GroupContentsResource & DispatchProp<any>) => renderName(resource));

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

export const renderDate = (date?: string) => {
    return (
        <Typography
            noWrap
            style={{ minWidth: "100px" }}
        >
            {date ? formatDate(date) : '-'}
        </Typography>
    );
};

export const renderWorkflowName = (item: WorkflowResource) => (
    <Grid
        container
        alignItems="center"
        wrap="nowrap"
        spacing={2}
    >
        <Grid item>{renderIcon(item)}</Grid>
        <Grid item>
            <Typography
                color="primary"
                style={{ width: "100px" }}
            >
                {item.name}
            </Typography>
        </Grid>
    </Grid>
);

// export const ResourceWorkflowName = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
//     return resource;
// })(renderWorkflowName);

const getPublicUuid = (uuidPrefix: string) => {
    return `${uuidPrefix}-tpzed-anonymouspublic`;
};

const resourceShare = (dispatch: Dispatch, uuidPrefix: string, ownerUuid?: string, uuid?: string) => {
    const isPublic = ownerUuid === getPublicUuid(uuidPrefix);
    return (
        <div>
            {!isPublic && uuid && (
                <Tooltip title="Share">
                    <IconButton onClick={() => dispatch<any>(openSharingDialog(uuid))} size="large">
                        <ShareIcon />
                    </IconButton>
                </Tooltip>
            )}
        </div>
    );
};

export const ResourceShare = connect((state: RootState, props: { resource: WorkflowResource }) => {
    const { resource } = props;
    const uuidPrefix = getUuidPrefix(state);
    return {
        uuid: resource ? resource.uuid : "",
        ownerUuid: resource ? resource.ownerUuid : "",
        uuidPrefix,
    };
})((props: { ownerUuid?: string; uuidPrefix: string; uuid?: string } & DispatchProp<any>) =>
    resourceShare(props.dispatch, props.uuidPrefix, props.ownerUuid, props.uuid)
);

// User Resources
// const renderFirstName = (item: { firstName: string }) => {
//     return <Typography noWrap>{item.firstName}</Typography>;
// };

// export const ResourceFirstName = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<UserResource>(props.uuid)(state.resources);
//     return resource || { firstName: "" };
// })(renderFirstName);

// const renderLastName = (item: { lastName: string }) => <Typography noWrap>{item.lastName}</Typography>;

// export const ResourceLastName = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<UserResource>(props.uuid)(state.resources);
//     return resource || { lastName: "" };
// })(renderLastName);

export const renderFullName = (item: { uuid: string; firstName: string; lastName: string }, link?: boolean) => {
    const displayName = (item.firstName + " " + item.lastName).trim() || item.uuid;
    return link ? (
        <Typography
            noWrap
            color="primary"
            style={{ cursor: "pointer" }}
            onClick={() => dispatchAction<any>(navigateToUserProfile, item.uuid)} 
        >
            {displayName}
        </Typography>
    ) : (
        <Typography noWrap>{displayName}</Typography>
    );
};

// export const UserResourceFullName = connect((state: RootState, props: { uuid: string; link?: boolean }) => {
//     const resource = getResource<UserResource>(props.uuid)(state.resources);
//     return { item: resource || { uuid: "", firstName: "", lastName: "" }, link: props.link };
// })((props: { item: { uuid: string; firstName: string; lastName: string }; link?: boolean } & DispatchProp<any>) =>
//     renderFullName(props.item, props.link)
// );

export const renderUuidWithCopy = (item: { uuid: string }) => (
    <Typography
        data-cy="uuid"
        noWrap
    >
        {item.uuid}
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || "-"}
    </Typography>
);

// export const renderResourceUuid = (resource: GroupContentsResource | GroupResource | UserResource) => (
//     <Typography
//         data-cy="uuid"
//         noWrap
//     >
//         {resource.uuid}
//         {(resource.uuid && <CopyToClipboardSnackbar value={resource.uuid} />) || "-"}
//     </Typography>
// );

// const renderUuidCopyIcon = (item: { uuid: string }) => (
//     <Typography
//         data-cy="uuid"
//         noWrap
//     >
//         {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || "-"}
//     </Typography>
// );

// export const ResourceUuid = connect(
//     (state: RootState, props: { uuid: string }) => getResource<UserResource>(props.uuid)(state.resources) || { uuid: "" }
// )(renderUuid);

export const renderEmail = (item: { email: string }) => <Typography noWrap>{item.email}</Typography>;

// export const ResourceEmail = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<UserResource>(props.uuid)(state.resources);
//     return resource || { email: "" };
// })(renderEmail);

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
        spacing={1}
        data-cy="account-status"
    >
        <Grid item>
            {(() => {
                switch (props.status) {
                    case UserAccountStatus.ACTIVE:
                        return <ActiveIcon style={{ color: "#4caf50", verticalAlign: "middle" }} />;
                    case UserAccountStatus.SETUP:
                        return <SetupIcon style={{ color: "#2196f3", verticalAlign: "middle" }} />;
                    case UserAccountStatus.INACTIVE:
                        return <InactiveIcon style={{ color: "#9e9e9e", verticalAlign: "middle" }} />;
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

export const ResourceLinkTailAccountStatus = connect((state: RootState, props: { resource: LinkResource }) => {
    return props.resource && props.resource.tailKind === ResourceKind.USER ? getUserAccountStatus(state, { uuid: props.resource.tailUuid }) : { status: UserAccountStatus.UNKNOWN };
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
                data-cy="user-visible-checkbox"
                color="primary"
                checked={props.visible}
                disabled={!props.canManage}
                onClick={e => {
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
    (state: RootState, props: { resource: LinkResource }) => {
        const member = getResource<Resource>(props.resource?.tailUuid || "")(state.resources);
        const group = getResource<GroupResource>(props.resource?.headUuid || "")(state.resources);
        const permissions = filterResources((resource: LinkResource) => {
            return (
                resource.linkClass === LinkClass.PERMISSION &&
                resource.headUuid === props.resource?.tailUuid &&
                resource.tailUuid === group?.uuid &&
                resource.name === PermissionLevel.CAN_READ
            );
        })(state.resources);

        const permissionLinkUuid = permissions.length > 0 ? permissions[0].uuid : "";
        const isVisible = props.resource && group && permissions.length > 0;
        // Consider whether the current user canManage this resurce in addition when it's possible
        const isBuiltin = isBuiltinGroup(props.resource?.headUuid || "");

        return member?.kind === ResourceKind.USER
            ? { memberLinkUuid: props.resource?.uuid, permissionLinkUuid, visible: isVisible, canManage: !isBuiltin }
            : { memberLinkUuid: "", permissionLinkUuid: "", visible: false, canManage: false };
    },
    { setMemberIsHidden }
)(renderIsHidden);

const renderIsAdmin = (props: { uuid: string; isAdmin: boolean; toggleIsAdmin: (uuid: string) => void }) => (
    <Checkbox
        color="primary"
        checked={props.isAdmin}
        onClick={e => {
            e.stopPropagation();
            props.toggleIsAdmin(props.uuid);
        }}
    />
);

export const ResourceIsAdmin = connect(
    (state: RootState, props: { resource: UserResource }) => {
        return props.resource || { isAdmin: false };
    },
    { toggleIsAdmin }
)(renderIsAdmin);

export const renderUsername = (item: { username: string; uuid: string }) => <Typography noWrap>{item.username || item.uuid}</Typography>;

// export const ResourceUsername = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<UserResource>(props.uuid)(state.resources);
//     return resource || { username: "", uuid: props.uuid };
// })(renderUsername);

// Virtual machine resource

const renderHostname = (item: { hostname: string }) => <Typography noWrap>{item.hostname}</Typography>;

export const VirtualMachineHostname = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<VirtualMachinesResource>(props.uuid)(state.resources);
    return resource || { hostname: "" };
})(renderHostname);

const renderVirtualMachineLogin = (login: { user: string }) => <Typography noWrap>{login.user}</Typography>;

export const VirtualMachineLogin = connect((state: RootState, props: { linkUuid: string }) => {
    const permission = getResource<LinkResource>(props.linkUuid)(state.resources);
    const user = getResource<UserResource>(permission?.tailUuid || "")(state.resources);

    return { user: user?.username || permission?.tailUuid || "" };
})(renderVirtualMachineLogin);

// Common methods
export const renderString = (str: string) => <Typography noWrap>{str || '-'}</Typography>;

export const renderUuid = (item: {uuid: string}) => <Typography noWrap>{item.uuid || '-'}</Typography>;

// const renderCommonDate = (date: string) => <Typography noWrap>{formatDate(date)}</Typography>;

// export const CommonUuid = withResourceData("uuid", renderCommonData);

// Api Client Authorizations
// export const TokenApiToken = withResourceData("apiToken", renderString);

// export const TokenCreatedByIpAddress = withResourceData("createdByIpAddress", renderDate);

// export const TokenExpiresAt = withResourceData("expiresAt", renderDate);

// export const TokenLastUsedAt = withResourceData("lastUsedAt", renderDate);

// export const TokenLastUsedByIpAddress = withResourceData("lastUsedByIpAddress", renderString);

// export const TokenScopes = withResourceData("scopes", renderString);

// export const TokenUserId = withResourceData("userId", renderString);

export const ResourceCluster = connect((state: RootState, props: { uuid: string }) => {
    const clusterId = props.uuid.slice(0, 5) || ""
    const clusterBadge = getProperty<ClusterBadge[]>('clusterBadges')(state.properties)?.find(badge => badge.text === clusterId);
    // dark grey is default BG color
    return clusterBadge || { text: clusterId, color: '#fff', backgroundColor: '#696969' };
})(renderClusterBadge);

function renderClusterBadge(badge: ClusterBadge) {
    
    const style = {
        backgroundColor: badge.backgroundColor,
        color: badge.color,
        padding: "2px 7px",
        borderRadius: 3,
    };

    return <span style={style}>{badge.text}</span>
};

// Links Resources
export const renderLinkName = (item: { name: string }) => <Typography noWrap>{item.name || "-"}</Typography>;

// export const ResourceLinkName = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<LinkResource>(props.uuid)(state.resources);
//     return resource || { name: "" };
// })(renderLinkName);

export const renderLinkClass = (item: { linkClass: string }) => <Typography noWrap>{item.linkClass}</Typography>;

// export const ResourceLinkClass = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<LinkResource>(props.uuid)(state.resources);
//     return resource || { linkClass: "" };
// })(renderLinkClass);

const getResourceDisplayName = (resource: Resource): string => {
    if ((resource as UserResource).kind === ResourceKind.USER && typeof (resource as UserResource).firstName !== "undefined") {
        // We can be sure the resource is UserResource
        return getUserDisplayName(resource as UserResource);
    } else {
        return (resource as GroupContentsResource).name;
    }
};

const renderResourceLink = (item: Resource ) => {
    var displayName = getResourceDisplayName(item);

    return (
        <Typography
            noWrap
            color="primary"
            style={{ cursor: "pointer" }}
            onClick={() => {
                item.kind === ResourceKind.GROUP && (item as GroupResource).groupClass === "role"
                    ? dispatchAction<any>(navigateToGroupDetails, item.uuid)
                    : item.kind === ResourceKind.USER
                    ? dispatchAction<any>(navigateToUserProfile, item.uuid)
                    : dispatchAction<any>(navigateTo, item.uuid); 
            }}
        >
            {resourceLabel(item.kind, item && item.kind === ResourceKind.GROUP ? (item as GroupResource).groupClass || "" : "")}:{" "}
            {displayName || item.uuid}
        </Typography>
    );
};

export const ResourceLinkTail = connect((state: RootState, props: { resource: PermissionResource | LinkResource }) => {
    const tailResource = getResource<Resource>(props.resource?.tailUuid || "")(state.resources);

    return {
        item: tailResource || { uuid: props.resource?.tailUuid || "", kind: props.resource?.tailKind || ResourceKind.NONE },
    };
})((props: { item: Resource } & DispatchProp<any>) => renderResourceLink(props.item));

export const ResourceLinkHead = connect((state: RootState, props: { resource: PermissionResource | LinkResource }) => {
    const headResource = getResource<Resource>(props.resource?.headUuid || "")(state.resources);
    return {
        item: headResource || { uuid: props.resource?.headUuid || "", kind: props.resource?.headKind || ResourceKind.NONE },
    };
})((props: { item: Resource } & DispatchProp<any>) => renderResourceLink(props.item));

// export const ResourceLinkUuid = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<LinkResource>(props.uuid)(state.resources);
//     return resource || { uuid: "" };
// })(renderUuid);

export const ResourceLinkHeadUuid = connect((state: RootState, props: { resource: PermissionResource }) => {
    const { resource } = props;
    const headResource = getResource<Resource>(resource?.headUuid || "")(state.resources);

    return headResource || { uuid: "" };
})(renderUuidWithCopy);

export const ResourceLinkTailUuid = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
    const tailResource = getResource<Resource>(link?.tailUuid || "")(state.resources);

    return tailResource || { uuid: "" };
})(renderUuidWithCopy);

const renderLinkDelete = (item: LinkResource, canManage: boolean) => {
    if (item.uuid) {
        return canManage ? (
            <Typography noWrap>
                <IconButton
                    data-cy="resource-delete-button"
                    onClick={() => dispatchAction<any>(openRemoveGroupMemberDialog, item.uuid)}
                    size="large">
                    <RemoveIcon />
                </IconButton>
            </Typography>
        ) : (
            <Typography noWrap>
                <IconButton disabled data-cy="resource-delete-button" size="large">
                    <RemoveIcon />
                </IconButton>
            </Typography>
        );
    } else {
        return <Typography noWrap></Typography>;
    }
};

export const ResourceLinkDelete = connect((state: RootState, props: { resource: PermissionResource }) => {
    const isBuiltin = isBuiltinGroup(props.resource?.headUuid || "") || isBuiltinGroup(props.resource?.tailUuid || "");

    return {
        item: props.resource || { uuid: "", kind: ResourceKind.NONE },
        canManage: props.resource && getResourceLinkCanManage(state, props.resource) && !isBuiltin,
    };
})((props: { item: LinkResource; canManage: boolean } & DispatchProp<any>) => renderLinkDelete(props.item, props.canManage));

export const ResourceLinkTailEmail = connect((state: RootState, props: { uuid: string }) => {
    const link = getResource<LinkResource>(props.uuid)(state.resources);
    const resource = getResource<UserResource>(link?.tailUuid || "")(state.resources);

    return resource || { email: "" };
})(renderEmail);

export const ResourceLinkTailUsername = connect((state: RootState, props: { resource: PermissionResource }) => {
    const resource = getResource<UserResource>(props.resource.tailUuid || "")(state.resources);
    return resource;
})((user:UserResource) => <Typography noWrap>{user.username || user.uuid || "-"}</Typography>);

const renderPermissionLevel = (link: LinkResource, canManage: boolean) => {
    return (
        <Typography noWrap>
            {formatPermissionLevel(link.name as PermissionLevel)}
            {canManage ? (
                <IconButton
                    data-cy="edit-permission-button"
                    onClick={event => dispatchAction<any>(openPermissionEditContextMenu, event, link)}
                    size="large">
                    <RenameIcon />
                </IconButton>
            ) : (
                ""
            )}
        </Typography>
    );
};

export const ResourceLinkHeadPermissionLevel = connect((state: RootState, props: { resource: PermissionResource }) => {
    const { resource } = props;
    const isBuiltin = isBuiltinGroup(resource?.headUuid || "") || isBuiltinGroup(resource?.tailUuid || "");

    return {
        link: resource || { uuid: "", name: "", kind: ResourceKind.NONE },
        canManage: resource && getResourceLinkCanManage(state, resource) && !isBuiltin,
    };
})((props: { link: LinkResource; canManage: boolean } & DispatchProp<any>) => renderPermissionLevel(props.link, props.canManage));

export const ResourceLinkTailPermissionLevel = connect((state: RootState, props: { resource: PermissionResource }) => {
    const isBuiltin = isBuiltinGroup(props.resource?.headUuid || "") || isBuiltinGroup(props.resource?.tailUuid || "");

    return {
        link: props.resource || { uuid: "", name: "", kind: ResourceKind.NONE },
        canManage: props.resource && getResourceLinkCanManage(state, props.resource) && !isBuiltin,
    };
})((props: { link: LinkResource; canManage: boolean } & DispatchProp<any>) => renderPermissionLevel(props.link, props.canManage));

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
export const resourceRunProcess = (uuid: string) => {
    return (
        <div>
            {uuid && (
                <Tooltip title="Run process">
                    <IconButton onClick={() => dispatchAction<any>(openRunProcess, uuid ?? '')} size="large">
                        <ProcessIcon />
                    </IconButton>
                </Tooltip>
            )}
        </div>
    );
};

// export const ResourceRunProcess = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
//     return {
//         uuid: resource ? resource.uuid : "",
//     };
// })((props: { uuid: string } & DispatchProp<any>) => resourceRunProcess(props.uuid));

const renderWorkflowStatus = (uuidPrefix: string, ownerUuid?: string) => {
    if (ownerUuid === getPublicUuid(uuidPrefix)) {
        return renderStatus(WorkflowStatus.PUBLIC);
    } else {
        return renderStatus(WorkflowStatus.PRIVATE);
    }
};

const renderStatus = (status: string) => (
    <Typography
        noWrap
        style={{ width: "60px" }}
    >
        {status}
    </Typography>
);

export const ResourceWorkflowStatus = connect((state: RootState, props: { resource: WorkflowResource }) => {
    const { resource } = props;
    const uuidPrefix = getUuidPrefix(state);
    return {
        ownerUuid: resource ? resource.ownerUuid : "",
        uuidPrefix,
    };
})((props: { ownerUuid?: string; uuidPrefix: string }) => renderWorkflowStatus(props.uuidPrefix, props.ownerUuid));

export const renderContainerUuid = (resource: GroupContentsResource) => {
    if (resource.kind !== ResourceKind.PROCESS) {
        return <>-</>;
    }
    const containerUuid = resource.containerUuid || '';
    return renderUuidWithCopy({ uuid: containerUuid });
};

// export const ResourceContainerUuid = connect((state: RootState, props: { uuid: string }) => {
//     const process = getProcess(props.uuid)(state.resources);
//     return { uuid: process?.container?.uuid ? process?.container?.uuid : "" };
// })((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

enum ColumnSelection {
    OUTPUT_UUID = "outputUuid",
    LOG_UUID = "logUuid",
}

const renderUuidLinkWithCopyIcon = (item: ProcessResource, column: string) => {
    const selectedColumnUuid = item[column];
    return (
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
                        onClick={() => dispatchAction<any>(navigateTo(selectedColumnUuid))}
                    >
                        {selectedColumnUuid && renderUuidWithCopy({ uuid: selectedColumnUuid })}
                    </Typography>
                ) : (
                    "-"
                )}
            </Grid>
        </Grid>
    );
};

export const renderResourceOutputUuid = (resource: GroupContentsResource) => {
    return resource.kind === ResourceKind.CONTAINER_REQUEST ? renderUuidLinkWithCopyIcon(resource, ColumnSelection.OUTPUT_UUID) : <>-</>;
}

// export const ResourceOutputUuid = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<ProcessResource>(props.uuid)(state.resources);
//     return resource;
// })((process: ProcessResource & DispatchProp<any>) => renderUuidLinkWithCopyIcon(process, ColumnSelection.OUTPUT_UUID));

export const renderResourceLogUuid = (resource: GroupContentsResource) => {
    return resource.kind === ResourceKind.CONTAINER_REQUEST ? renderUuidLinkWithCopyIcon(resource, ColumnSelection.LOG_UUID) : <>-</>;
}

// export const ResourceLogUuid = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<ProcessResource>(props.uuid)(state.resources);
//     return resource;
// })((process: ProcessResource & DispatchProp<any>) => renderUuidLinkWithCopyIcon(process, ColumnSelection.LOG_UUID));

export const renderResourceParentProcess = (resource: GroupContentsResource) => {
    return resource.kind === ResourceKind.CONTAINER_REQUEST ? renderUuidWithCopy({ uuid: (resource as ContainerRequestResource).requestingContainerUuid || "" }) : <>-</>;
}

// export const ResourceParentProcess = connect((state: RootState, props: { uuid: string }) => {
//     const process = getProcess(props.uuid)(state.resources);
//     return { parentProcess: process?.containerRequest?.requestingContainerUuid || "" };
// })((props: { parentProcess: string }) => renderUuid({ uuid: props.parentProcess }));

// export const ResourceModifiedByUserUuid = connect((state: RootState, props: { uuid: string }) => {
//     const process = getProcess(props.uuid)(state.resources);
//     return { userUuid: process?.containerRequest?.modifiedByUserUuid || "" };
// })((props: { userUuid: string }) => renderUuid({ uuid: props.userUuid }));

export const renderModifiedByUserUuid = (resource: GroupContentsResource & {containerRequest?: any}) => {
    const modifiedByUserUuid = resource.containerRequest ? resource.containerRequest.modifiedByUserUuid : resource.modifiedByUserUuid;
    return renderUuidWithCopy({uuid:modifiedByUserUuid});
}

export const renderCreatedAtDate = (resource: GroupContentsResource) => {
    return renderDate(resource.createdAt);
}

// export const ResourceCreatedAtDate = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
//     return { date: resource ? resource.createdAt : "" };
// })((props: { date: string }) => renderDate(props.date));

export const renderLastModifiedDate = (resource: GroupContentsResource) => {
    return renderDate(resource.modifiedAt);
}

// export const ResourceLastModifiedDate = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
//     return { date: resource ? resource.modifiedAt : "" };
// })((props: { date: string }) => renderDate(props.date));

export const renderTrashDate = (resource: TrashableResource) => {
    return renderDate(resource.trashAt);
}

// export const ResourceTrashDate = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<TrashableResource>(props.uuid)(state.resources);
//     return { date: resource ? resource.trashAt : "" };
// })((props: { date: string }) => renderDate(props.date));

export const renderDeleteDate = (resource: TrashableResource) => {
    return renderDate(resource.deleteAt);
}

// export const ResourceDeleteDate = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<TrashableResource>(props.uuid)(state.resources);
//     return { date: resource ? resource.deleteAt : "" };
// })((props: { date: string }) => renderDate(props.date));

export const renderFileSize = (resource: GroupContentsResource & { fileSizeTotal?: number }) => (
    <Typography
        noWrap
        style={{ minWidth: "45px" }}
    >
        {formatFileSize(resource.fileSizeTotal)}
    </Typography>
);

// export const ResourceFileSize = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<CollectionResource>(props.uuid)(state.resources);

//     if (resource && resource.kind !== ResourceKind.COLLECTION) {
//         return { fileSize: "" };
//     }

//     return { fileSize: resource ? resource.fileSizeTotal : 0 };
// })((props: { fileSize?: number }) => renderFileSize(props.fileSize));

// const renderOwner = (owner: string) => <Typography noWrap>{owner || "-"}</Typography>;

// export const ResourceOwner = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
//     return { owner: resource ? resource.ownerUuid : "" };
// })((props: { owner: string }) => renderOwner(props.owner));

// export const ResourceOwnerName = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
//     const ownerNameState = state.ownerName;
//     const ownerName = ownerNameState.find(it => it.uuid === resource!.ownerUuid);
//     return { owner: ownerName ? ownerName!.name : resource!.ownerUuid };
// })((props: { owner: string }) => renderOwner(props.owner));

// export const ResourceUUID = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<CollectionResource>(props.uuid)(state.resources);
//     return { uuid: resource ? resource.uuid : "" };
// })((props: { uuid: string }) => renderUuid({ uuid: props.uuid }));

export const renderVersion = (resource: CollectionResource) => {
    return <Typography>{resource.version ?? "-"}</Typography>;
};

// export const ResourceVersion = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<CollectionResource>(props.uuid)(state.resources);
//     return { version: resource ? resource.version : "" };
// })((props: { version: number }) => renderVersion(props.version));

export const renderPortableDataHash = (resource: GroupContentsResource) => (
    <Typography noWrap>
        {'portableDataHash' in resource ? (
            <>
                {resource.portableDataHash}
                <CopyToClipboardSnackbar value={resource.portableDataHash} />
            </>
        ) : (
            "-"
        )}
    </Typography>
);

// export const ResourcePortableDataHash = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<CollectionResource>(props.uuid)(state.resources);
//     return { portableDataHash: resource ? resource.portableDataHash : "" };
// })((props: { portableDataHash: string }) => renderPortableDataHash(props.portableDataHash));

export const renderFileCount = (resource: GroupContentsResource & { fileCount?: number }) => {
    return <Typography>{resource.fileCount ?? "-"}</Typography>;
};

// export const ResourceFileCount = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<CollectionResource>(props.uuid)(state.resources);
//     return { fileCount: resource ? resource.fileCount : "" };
// })((props: { fileCount: number }) => renderFileCount(props.fileCount));

const userFromID = connect((state: RootState, props: { uuid: string }) => {
    let userFullname = "";
    const resource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);

    if (resource) {
        userFullname = getUserFullname(resource as User) || (resource as GroupContentsResource).name;
    }

    return { uuid: props.uuid, userFullname };
});

// const ownerFromResourceId = compose(
//     connect((state: RootState, props: { uuid: string }) => {
//         const childResource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);
//         return { uuid: childResource ? (childResource as Resource).ownerUuid : "" };
//     }),
//     userFromID
// );

const _resourceWithName = withStyles(
    {},
    { withTheme: true }
)((props: { uuid: string; userFullname: string; dispatch: Dispatch; theme: ArvadosTheme }) => {
    const { uuid, userFullname, dispatch, theme } = props;
    if (userFullname === "") {
        dispatch<any>(loadResource(uuid, false));
        return (
            <Typography
                style={{ color: theme.palette.primary.main }}
                display="inline"
            >
                {uuid}
            </Typography>
        );
    }

    return (
        <Typography
            style={{ color: theme.palette.primary.main }}
            display="inline"
        >
            {userFullname} ({uuid})
        </Typography>
    );
});

// const _resourceWithNameLink = withStyles(
//     {},
//     { withTheme: true }
// )((props: { uuid: string; userFullname: string; dispatch: Dispatch; theme: ArvadosTheme }) => {
//     const { uuid, userFullname, dispatch, theme } = props;
//     if (!userFullname) {
//         dispatch<any>(loadResource(uuid, false));
//     }

//     return (
//         <Typography
//             style={{ color: theme.palette.primary.main, cursor: 'pointer' }}
//             display="inline"
//             noWrap
//             onClick={() => dispatch<any>(navigateTo(uuid))}
//         >
//             {userFullname ? userFullname : uuid}
//         </Typography>
//     )
// });


// export const ResourceOwnerWithNameLink = ownerFromResourceId(_resourceWithNameLink);

// export const ResourceOwnerWithName = ownerFromResourceId(_resourceWithName);

export const OwnerWithName = connect((state: RootState, props: { resource: GroupContentsResource; link?: boolean }) => {
    const owner = getResource<UserResource>(props.resource.ownerUuid)(state.resources);
    const ownerName = owner ? getUserDisplayName(owner) : props.resource.ownerUuid;
    return { ownerName, ownerUuid: props.resource.ownerUuid, link: props.link };
})((props: { ownerName: string; ownerUuid: string; link?: boolean }) => {
    return props.link ? (
        <Typography
            style={{ color: CustomTheme.palette.primary.main, cursor: 'pointer' }}
            display='inline'
            noWrap
            onClick={() => dispatchAction<any>(navigateTo(props.ownerUuid))}
        >
            {props.ownerName ? props.ownerName : props.ownerUuid}
        </Typography>
    ) : (
        <Typography
            noWrap
            style={{ color: CustomTheme.palette.primary.main }}
            display='inline'
        >
            {props.ownerName ? props.ownerName : props.ownerUuid}
        </Typography>
    );
});


export const ResourceWithName = userFromID(_resourceWithName);

export const UserNameFromID = compose(userFromID)((props: { uuid: string; displayAsText?: string; userFullname: string; dispatch: Dispatch }) => {
    const { uuid, userFullname, dispatch } = props;

    if (userFullname === "") {
        dispatch<any>(loadResource(uuid, false));
    }
    return <span>{userFullname ? userFullname : uuid}</span>;
});

export const ResponsiblePerson = compose(
    connect((state: RootState, props: { uuid: string; parentRef: HTMLElement | null }) => {
        let responsiblePersonName: string = "";
        let responsiblePersonUUID: string = "";
        let responsiblePersonProperty: string = "";

        if (state.auth.config.clusterConfig.Collections.ManagedProperties) {
            let index = 0;
            const keys = Object.keys(state.auth.config.clusterConfig.Collections.ManagedProperties);

            while (!responsiblePersonProperty && keys[index]) {
                const key = keys[index];
                if (state.auth.config.clusterConfig.Collections.ManagedProperties[key].Function === "original_owner") {
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
        parentRef.style.display = "none";
        return null;
    } else if (parentRef) {
        parentRef.style.display = "block";
    }

    if (!responsiblePersonName) {
        return (
            <Typography
                style={{ color: theme.palette.primary.main }}
                display="inline"
                noWrap
            >
                {uuid}
            </Typography>
        );
    }

    return (
        <Typography
            style={{ color: theme.palette.primary.main }}
            display="inline"
            noWrap
        >
            {responsiblePersonName} ({uuid})
        </Typography>
    );
});

export const renderType = (resource: GroupContentsResource | undefined) => {
    if(!resource) return <Typography noWrap>-</Typography>;
    const type = resource.kind;
    const subtype = resource.kind === ResourceKind.GROUP
                        ? resource.groupClass
                        : resource.kind === ResourceKind.PROCESS
                            ? resource.requestingContainerUuid
                                ? ProcessTypeFilter.CHILD_PROCESS
                                : ProcessTypeFilter.MAIN_PROCESS
                            : ""
    return<Typography noWrap>{resourceLabel(type, subtype)}</Typography>
};

// export const ResourceType = connect((state: RootState, props: { uuid: string }) => {
//     const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
//     return resource
//     // return {
//     //     type: resource ? resource.kind : "",
//     //     subtype: resource
//     //         ? resource.kind === ResourceKind.GROUP
//     //             ? resource.groupClass
//     //             : resource.kind === ResourceKind.PROCESS
//     //                 ? resource.requestingContainerUuid
//     //                     ? ProcessTypeFilter.CHILD_PROCESS
//     //                     : ProcessTypeFilter.MAIN_PROCESS
//     //                 : ""
//     //         : ""
//     // };
// })((props: { resource: any}) => renderType(props.resource || undefined));

export const renderResourceStatus = (resource: GroupContentsResource) => {
    return resource.kind === ResourceKind.COLLECTION ? <CollectionStatus collection={resource} /> : <ProcessStatus uuid={resource.uuid} />;
}

// export const ResourceStatus = connect((state: RootState, props: { uuid: string }) => {
//     return { resource: getResource<GroupContentsResource>(props.uuid)(state.resources) };
// })((props: { resource: GroupContentsResource }) =>
//     props.resource && props.resource.kind === ResourceKind.COLLECTION ? (
//         <CollectionStatus uuid={props.resource.uuid} />
//     ) : (
//         <ProcessStatus uuid={props.resource.uuid} />
//     )
// );

export const CollectionStatus = (props: { collection: CollectionResource }) =>
    props.collection.uuid !== props.collection.currentVersionUuid ? (
        <Typography>version {props.collection.version}</Typography>
    ) : (
        <Typography>head version</Typography>
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
            data-cy="process-status-chip"
            label={getProcessStatus(props.process)}
            style={{
                height: props.theme.spacing(3),
                width: props.theme.spacing(12),
                ...getProcessStatusStyles(getProcessStatus(props.process), props.theme),
                fontSize: "0.875rem",
                borderRadius: props.theme.spacing(0.625),
            }}
        />
    ) : (
        <Typography>-</Typography>
    )
);

// export const ProcessStartDate = connect((state: RootState, props: { uuid: string }) => {
//     const process = getProcess(props.uuid)(state.resources);
//     return { date: process && process.container ? process.container.startedAt : "" };
// })((props: { date: string }) => renderDate(props.date));

export const renderRunTime = (time: number) => (
    <Typography
        noWrap
        style={{ minWidth: "45px" }}
    >
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

// export const GroupMembersCount = connect(
//     (state: RootState, props: { uuid: string }) => {
//         const group = getResource<GroupResource>(props.uuid)(state.resources);

//         return {
//             value: group?.memberCount,
//         };

//     }
// )(withTheme((props: {value: number | null | undefined, theme:ArvadosTheme}) => {
//     if (props.value === undefined) {
//         // Loading
//         return <Typography component={"div"}>
//             <InlinePulser />
//         </Typography>;
//     } else if (props.value === null) {
//         // Error
//         return <Typography>
//             <Tooltip title="Failed to load member count">
//                 <ErrorIcon style={{color: props.theme.customs.colors.greyL}}/>
//             </Tooltip>
//         </Typography>;
//     } else {
//         return <Typography children={props.value} />;
//     }
// }));

export const renderMembersCount = (resource: GroupResource) => {
    const value = resource.memberCount;
    if (value === undefined) {
        // Loading
        return <Typography component={"div"}>
            <InlinePulser />
        </Typography>;
    } else if (value === null) {
        // Error
        return <Typography>
            <Tooltip title="Failed to load member count">
                <ErrorIcon style={{color: CustomTheme.palette.grey['600']}}/>
            </Tooltip>
        </Typography>;
    } else {
        return <Typography children={value} />;
    }
};

export const renderRestoreFromTrash = (resource: TrashableResource) => {
    return (
        <Tooltip title="Restore">
            <IconButton
                style={{ padding: '0' }}
                onClick={() => {
                    if (resource) {
                        dispatchAction(toggleTrashed,
                            resource.kind,
                            resource.uuid,
                            resource.ownerUuid,
                            resource.isTrashed
                        );
                    }}}
                size="large">
                <RestoreFromTrashIcon />
            </IconButton>
        </Tooltip>
    );
};
