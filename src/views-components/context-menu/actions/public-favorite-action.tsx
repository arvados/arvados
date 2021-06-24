// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { PublicFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";

const mapStateToProps = (state: RootState, props: { onClick: () => {} }) => ({
    isPublicFavorite: state.contextMenu.resource !== undefined && state.publicFavorites[state.contextMenu.resource.uuid] === true,
    onClick: props.onClick
});

export const TogglePublicFavoriteAction = connect(mapStateToProps)((props: { isPublicFavorite: boolean, onClick: () => void }) =>
    <ListItem
        button
        onClick={props.onClick}>
        <ListItemIcon>
            {props.isPublicFavorite
                ? <PublicFavoriteIcon />
                : <PublicFavoriteIcon />}
        </ListItemIcon>
        <ListItemText style={{ textDecoration: 'none' }}>
            {props.isPublicFavorite
                ? <>Remove from public favorites</>
                : <>Add to public favorites</>}
        </ListItemText>
    </ListItem >);
