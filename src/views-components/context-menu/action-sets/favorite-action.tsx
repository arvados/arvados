// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ListItemIcon, ListItemText } from "@material-ui/core";
import { FavoriteIcon, AddFavoriteIcon, RemoveFavoriteIcon } from "../../../components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "../../../store/store";

const mapStateToProps = (state: RootState) => ({
    isFavorite: state.contextMenu.resource && state.favorites[state.contextMenu.resource.uuid] === true
});

export const FavoriteActionText = connect(mapStateToProps)((props: { isFavorite: boolean }) =>
    props.isFavorite
        ? <>Remove from favorites</>
        : <>Add to favorites</>);

export const FavoriteActionIcon = connect(mapStateToProps)((props: { isFavorite: boolean }) =>
    props.isFavorite
        ? <RemoveFavoriteIcon />
        : <AddFavoriteIcon />);
