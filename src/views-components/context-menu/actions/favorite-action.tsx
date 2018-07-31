// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ListItemIcon, ListItemText } from "@material-ui/core";
import { AddFavoriteIcon, RemoveFavoriteIcon } from "../../../components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "../../../store/store";

const mapStateToProps = (state: RootState) => ({
    isFavorite: state.contextMenu.resource && state.favorites[state.contextMenu.resource.uuid] === true
});

export const ToggleFavoriteAction = connect(mapStateToProps)((props: { isFavorite: boolean }) =>
    <>
        <ListItemIcon>
            {props.isFavorite
                ? <RemoveFavoriteIcon />
                : <AddFavoriteIcon />}
        </ListItemIcon>
        <ListItemText>
            {props.isFavorite
                ? <>Remove from favorites</>
                : <>Add to favorites</>}
        </ListItemText>
    </>);
