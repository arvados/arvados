// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { RestoreFromTrashIcon, TrashIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";

const mapStateToProps = (state: RootState, props: { onClick: () => {} }) => ({
    isTrashed: state.contextMenu.resource && state.contextMenu.resource.isTrashed,
    onClick: props.onClick
});

export const ToggleTrashAction = connect(mapStateToProps)((props: { isTrashed?: boolean, onClick: () => void }) =>
    <ListItem button
        onClick={props.onClick}>
        <ListItemIcon>
            {props.isTrashed
                ? <RestoreFromTrashIcon/>
                : <TrashIcon/>}
        </ListItemIcon>
        <ListItemText style={{ textDecoration: 'none' }}>
            {props.isTrashed ? "Restore" : "Move to trash"}
        </ListItemText>
    </ListItem >);
