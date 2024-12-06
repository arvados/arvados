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
import { getProperty } from "store/properties/properties";
import { ClusterBadge } from "store/auth/cluster-badges";
import { PermissionResource } from 'models/permission';
import { ContainerRequestResource } from 'models/container-request';
import { toggleTrashed } from "store/trash/trash-actions";

// utility renderers ---------------------------------------------------------------------------------

export const renderString = (str: string) => <Typography noWrap>{str || '-'}</Typography>;

export const renderUuid = (item: {uuid: string}) => <Typography noWrap>{item.uuid || '-'}</Typography>;

export const renderUuidWithCopy = (item: { uuid: string }) => (
    <Typography
        data-cy="uuid"
        noWrap
    >
        {item.uuid}
        {(item.uuid && <CopyToClipboardSnackbar value={item.uuid} />) || "-"}
    </Typography>
);

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

export const renderCreatedAtDate = (resource: GroupContentsResource) => {
    return renderDate(resource.createdAt);
}

export const renderLastModifiedDate = (resource: GroupContentsResource) => {
    return renderDate(resource.modifiedAt);
}

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

export const renderResourceStatus = (resource: GroupContentsResource) => {
    return resource.kind === ResourceKind.COLLECTION ? <CollectionStatus collection={resource} /> : <ProcessStatus uuid={resource.uuid} />;
}

const renderIcon = (item: GroupContentsResource  | GroupResource) => {
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

const renderUuidLinkWithCopyIcon = (item: ProcessResource, column: string, dispatch: Dispatch) => {
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
                        onClick={() => dispatch<any>(navigateTo(selectedColumnUuid))}
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

export const RenderName = connect((resource: GroupContentsResource | GroupResource) => resource)(
    (props: { resource: GroupContentsResource | GroupResource } & DispatchProp<any>) => {
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
                    {resource.kind === ResourceKind.PROJECT && <FrozenProject item={resource as ProjectResource} />}
                </Typography>
            </Grid>
        </Grid>
    );
});

export const RenderOwnerName = connect((state: RootState, props: { resource: GroupContentsResource; link?: boolean }) => {
    const owner = getResource<any>(props.resource.ownerUuid)(state.resources);
    const ownerName = owner ? 'fullName' in owner ? getUserDisplayName(owner) : owner.name : null;
    return { ownerName, ownerUuid: props.resource.ownerUuid, link: props.link };
})((props: { ownerName: string; ownerUuid: string; link?: boolean } & DispatchProp<any>) => {
    return props.link ? (
        <Typography
            style={{ color: CustomTheme.palette.primary.main, cursor: 'pointer' }}
            display='inline'
            noWrap
            onClick={() => props.dispatch<any>(navigateTo(props.ownerUuid))}
        >
            {props.ownerName ? `${props.ownerName} (${props.ownerUuid})` : props.ownerUuid}
        </Typography>
    ) : (
        <Typography
            noWrap
            display='inline'
        >
            {props.ownerName ? `${props.ownerName} (${props.ownerUuid})` : props.ownerUuid}
        </Typography>
    );
});

// Project resource renderers ---------------------------------------------------------------------------------
const FrozenProject = (props: { item: ProjectResource }) => {
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

// User resource renderers ---------------------------------------------------------------------------------
export const renderUsername = (item: { username: string; uuid: string }) => <Typography noWrap>{item.username || item.uuid}</Typography>;

export const renderEmail = (item: { email: string }) => <Typography noWrap>{item.email}</Typography>;

export const RenderFullName = connect((resource: UserResource) => resource)((props: { resource: UserResource} & DispatchProp<any>) => {
    const { resource, dispatch } = props;
    const displayName = (resource.firstName + " " + resource.lastName).trim() || resource.uuid;
    return (
        <Typography
            noWrap
            color="primary"
            style={{ cursor: "pointer" }}
            onClick={() => dispatch<any>(navigateToUserProfile(resource.uuid))} 
        >
            {displayName}
        </Typography>
    )
});

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

const toggleIsAdmin = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        const isAdmin = data!.isAdmin;
        const newActivity = await services.userService.update(uuid, { isAdmin: !isAdmin });
        dispatch<any>(loadUsersPanel());
        return newActivity;
    };

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

// Permissions renderers ---------------------------------------------------------------------------------
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

// Virtual Machines renderers ---------------------------------------------------------------------------------

export const VirtualMachineHostname = connect((state: RootState, props: { uuid: string }) => {
    const resource = getResource<VirtualMachinesResource>(props.uuid)(state.resources);
    return resource || { hostname: "" };
})((item: { hostname: string }) => <Typography noWrap>{item.hostname}</Typography>);

export const VirtualMachineLogin = connect((state: RootState, props: { linkUuid: string }) => {
    const permission = getResource<LinkResource>(props.linkUuid)(state.resources);
    const user = getResource<UserResource>(permission?.tailUuid || "")(state.resources);

    return { user: user?.username || permission?.tailUuid || "" };
})((login: { user: string }) => <Typography noWrap>{login.user}</Typography>);

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

// Links renderers ---------------------------------------------------------------------------------
const getResourceDisplayName = (resource: Resource): string => {
    if ((resource as UserResource).kind === ResourceKind.USER && typeof (resource as UserResource).firstName !== "undefined") {
        // We can be sure the resource is UserResource
        return getUserDisplayName(resource as UserResource);
    } else {
        return (resource as GroupContentsResource).name;
    }
};

const renderResourceLink = (item: Resource , dispatch: Dispatch) => {
    var displayName = getResourceDisplayName(item);

    return (
        <Typography
            noWrap
            color="primary"
            style={{ cursor: "pointer" }}
            onClick={() => {
                item.kind === ResourceKind.GROUP && (item as GroupResource).groupClass === "role"
                    ? dispatch<any>(navigateToGroupDetails(item.uuid))
                    : item.kind === ResourceKind.USER
                    ? dispatch<any>(navigateToUserProfile(item.uuid))
                    : dispatch<any>(navigateTo(item.uuid)); 
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
})((props: { item: Resource } & DispatchProp<any>) => renderResourceLink(props.item, props.dispatch));

export const ResourceLinkHead = connect((state: RootState, props: { resource: PermissionResource | LinkResource }) => {
    const headResource = getResource<Resource>(props.resource?.headUuid || "")(state.resources);
    return {
        item: headResource || { uuid: props.resource?.headUuid || "", kind: props.resource?.headKind || ResourceKind.NONE },
    };
})((props: { item: Resource } & DispatchProp<any>) => renderResourceLink(props.item, props.dispatch));

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

const renderLinkDelete = (item: LinkResource, canManage: boolean, dispatch: Dispatch) => {
    if (item.uuid) {
        return canManage ? (
            <Typography noWrap>
                <IconButton
                    data-cy="resource-delete-button"
                    onClick={() => dispatch<any>(openRemoveGroupMemberDialog(item.uuid))}
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
})((props: { item: LinkResource; canManage: boolean } & DispatchProp<any>) => renderLinkDelete(props.item, props.canManage, props.dispatch));

export const ResourceLinkTailUsername = connect((state: RootState, props: { resource: PermissionResource }) => {
    const resource = getResource<UserResource>(props.resource.tailUuid || "")(state.resources);
    return resource;
})((user:UserResource) => <Typography noWrap>{user.username || user.uuid || "-"}</Typography>);

const renderPermissionLevel = (link: LinkResource, canManage: boolean, dispatch: Dispatch) => {
    return (
        <Typography noWrap>
            {formatPermissionLevel(link.name as PermissionLevel)}
            {canManage ? (
                <IconButton
                    data-cy="edit-permission-button"
                    onClick={event => dispatch<any>(openPermissionEditContextMenu(event, link))}
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
})((props: { link: LinkResource; canManage: boolean } & DispatchProp<any>) => renderPermissionLevel(props.link, props.canManage, props.dispatch));

export const ResourceLinkTailPermissionLevel = connect((state: RootState, props: { resource: PermissionResource }) => {
    const isBuiltin = isBuiltinGroup(props.resource?.headUuid || "") || isBuiltinGroup(props.resource?.tailUuid || "");

    return {
        link: props.resource || { uuid: "", name: "", kind: ResourceKind.NONE },
        canManage: props.resource && getResourceLinkCanManage(state, props.resource) && !isBuiltin,
    };
})((props: { link: LinkResource; canManage: boolean } & DispatchProp<any>) => renderPermissionLevel(props.link, props.canManage, props.dispatch));

const getResourceLinkCanManage = (state: RootState, link: LinkResource) => {
    const headResource = getResource<Resource>(link.headUuid)(state.resources);
    if (headResource && headResource.kind === ResourceKind.GROUP) {
        return (headResource as GroupResource).canManage;
    } else {
        // true for now
        return true;
    }
};

// Process / Workflow renderers ---------------------------------------------------------------------------------
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

const renderRunTime = (time: number) => (
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

export const ResourceRunProcess = connect((uuid: string) => uuid)((props: { uuid:string } & DispatchProp<any>) => {
    const { uuid } = props;
    return (
        <div>
            {uuid && (
                <Tooltip title="Run process">
                    <IconButton onClick={() => props.dispatch<any>(openRunProcess(uuid ?? ''))} size="large">
                        <ProcessIcon />
                    </IconButton>
                </Tooltip>
            )}
        </div>
    );
});

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

enum ColumnSelection {
    OUTPUT_UUID = "outputUuid",
    LOG_UUID = "logUuid",
}

export const ResourceOutputUuid = connect((state: RootState, props: { resource: ProcessResource }) => { 
    return {process: props.resource};
})((props: {process: ProcessResource} & DispatchProp<any>) => renderUuidLinkWithCopyIcon(props.process, ColumnSelection.OUTPUT_UUID, props.dispatch));

export const ResourceLogUuid = connect((state: RootState, props: { resource: ProcessResource }) => {
    return {process:props.resource};
})((props: {process: ProcessResource} & DispatchProp<any>) => renderUuidLinkWithCopyIcon(props.process, ColumnSelection.LOG_UUID, props.dispatch));

export const renderResourceParentProcess = (resource: GroupContentsResource) => {
    return resource.kind === ResourceKind.CONTAINER_REQUEST ? renderUuidWithCopy({ uuid: (resource as ContainerRequestResource).requestingContainerUuid || "" }) : <>-</>;
}

export const renderModifiedByUserUuid = (resource: GroupContentsResource & {containerRequest?: any}) => {
    const modifiedByUserUuid = resource.containerRequest ? resource.containerRequest.modifiedByUserUuid : resource.modifiedByUserUuid;
    return renderUuidWithCopy({uuid:modifiedByUserUuid});
}


// Collection renderers ---------------------------------------------------------------------------------
export const renderTrashDate = (resource: TrashableResource) => {
    return renderDate(resource.trashAt);
}

export const renderDeleteDate = (resource: TrashableResource) => {
    return renderDate(resource.deleteAt);
}

export const renderFileSize = (resource: GroupContentsResource & { fileSizeTotal?: number }) => (
    <Typography
        noWrap
        style={{ minWidth: "45px" }}
    >
        {formatFileSize(resource.fileSizeTotal)}
    </Typography>
);

export const renderVersion = (resource: CollectionResource) => {
    return <Typography>{resource.version ?? "-"}</Typography>;
};

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

export const renderFileCount = (resource: GroupContentsResource & { fileCount?: number }) => {
    return <Typography>{resource.fileCount ?? "-"}</Typography>;
};

const userFromID = connect((state: RootState, props: { uuid: string }) => {
    let userFullname = "";
    const resource = getResource<GroupContentsResource & UserResource>(props.uuid)(state.resources);

    if (resource) {
        userFullname = getUserFullname(resource as User) || (resource as GroupContentsResource).name;
    }

    return { uuid: props.uuid, userFullname };
});

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

const CollectionStatus = (props: { collection: CollectionResource }) =>
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

// Group renderers ---------------------------------------------------------------------------------
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

// Trash renderers ---------------------------------------------------------------------------------
export const RestoreFromTrash = connect((resource: TrashableResource | CollectionResource)=> resource)(
    (props: {resource: TrashableResource} & DispatchProp<any>) => {
    const { resource, dispatch } = props;
    return (
        <Tooltip title="Restore">
            <IconButton
                style={{ padding: '0' }}
                onClick={() => {
                    if (resource) {
                        dispatch<any>(toggleTrashed(
                            resource.kind,
                            resource.uuid,
                            resource.ownerUuid,
                            resource.isTrashed)
                        );
                    }}}
                size="large">
                <RestoreFromTrashIcon />
            </IconButton>
        </Tooltip>
    );
});
