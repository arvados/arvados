// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { LockIcon, UnlockIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { ProjectResource } from "models/project";
import { withRouter, RouteComponentProps } from "react-router";

const mapStateToProps = (state: RootState, props: { onClick: () => {} }) => ({
    isAdmin: state.auth.user!.isAdmin,
    isLocked: !!(state.resources[state.contextMenu.resource!.uuid] as ProjectResource).frozenByUuid,
    onClick: props.onClick
});

export const ToggleLockAction = withRouter(connect(mapStateToProps)((props: { isLocked: boolean, isAdmin: boolean, onClick: () => void } & RouteComponentProps) =>
    props.isLocked && !props.isAdmin ? null :
        < ListItem
            button
            onClick={props.onClick} >
            <ListItemIcon>
                {props.isLocked
                    ? <UnlockIcon />
                    : <LockIcon />}
            </ListItemIcon>
            <ListItemText style={{ textDecoration: 'none' }}>
                {props.isLocked
                    ? <>Unlock project</>
                    : <>Lock project</>}
            </ListItemText>
        </ListItem >));
