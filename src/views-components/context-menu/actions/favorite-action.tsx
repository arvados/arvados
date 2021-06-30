// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { AddFavoriteIcon, RemoveFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";

const mapStateToProps = (state: RootState, props: { onClick: () => {} }) => ({
    isFavorite: state.contextMenu.resource !== undefined && state.favorites[state.contextMenu.resource.uuid] === true,
    onClick: props.onClick
});

export const ToggleFavoriteAction = connect(mapStateToProps)((props: { isFavorite: boolean, onClick: () => void }) =>
    <ListItem
        button
        onClick={props.onClick}>
        <ListItemIcon>
            {props.isFavorite
                ? <RemoveFavoriteIcon />
                : <AddFavoriteIcon />}
        </ListItemIcon>
        <ListItemText style={{ textDecoration: 'none' }}>
            {props.isFavorite
                ? <>Remove from favorites</>
                : <>Add to favorites</>}
        </ListItemText>
    </ListItem >);
