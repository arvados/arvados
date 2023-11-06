// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { FreezeIcon, UnfreezeIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { ProjectResource } from "models/project";
import { withRouter, RouteComponentProps } from "react-router";
import { resourceIsFrozen } from "common/frozen-resources";

const mapStateToProps = (state: RootState, props: { onClick: () => {} }) => ({
    isAdmin: !!state.auth.user?.isAdmin,
    isLocked: !!(state.resources[state.contextMenu.resource!.uuid] as ProjectResource).frozenByUuid,
    canManage: (state.resources[state.contextMenu.resource!.uuid] as ProjectResource).canManage,
    canUnfreeze: !state.auth.remoteHostsConfig[state.auth.homeCluster]?.clusterConfig?.API?.UnfreezeProjectRequiresAdmin,
    resource: state.contextMenu.resource,
    resources: state.resources,
    onClick: props.onClick
});

export const ToggleLockAction = withRouter(connect(mapStateToProps)((props: {
    resource: any,
    resources: any,
    onClick: () => void,
    state: RootState, isAdmin: boolean, isLocked: boolean, canManage: boolean, canUnfreeze: boolean,
} & RouteComponentProps) =>
    (props.canManage && !props.isLocked) || (props.isLocked && props.canManage && (props.canUnfreeze || props.isAdmin))  ? 
        resourceIsFrozen(props.resource, props.resources) ? null :
            <ListItem
                button
                onClick={props.onClick} >
                <ListItemIcon>
                    {props.isLocked
                        ? <UnfreezeIcon />
                        : <FreezeIcon />}
                </ListItemIcon>
                <ListItemText style={{ textDecoration: 'none' }}>
                    {props.isLocked
                        ? <>Unfreeze project</>
                        : <>Freeze project</>}
                </ListItemText>
            </ListItem > : null));
