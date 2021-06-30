// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { FavoriteIcon, PublicFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { withStyles, StyleRulesCallback, WithStyles, Tooltip } from "@material-ui/core";

type CssRules = "icon";

const styles: StyleRulesCallback<CssRules> = theme => ({
    icon: {
        fontSize: "inherit"
    }
});

const mapStateToProps = (state: RootState, props: { resourceUuid: string; className?: string; }) => ({
    ...props,
    isFavoriteVisible: state.favorites[props.resourceUuid],
    isPublicFavoriteVisible: state.publicFavorites[props.resourceUuid]
});

export const FavoriteStar = connect(mapStateToProps)(
    withStyles(styles)((props: { isFavoriteVisible: boolean; className?: string; } & WithStyles<CssRules>) => {
        if (props.isFavoriteVisible) {
            return <Tooltip enterDelay={500} title="Favorite"><FavoriteIcon className={props.className || props.classes.icon} /></Tooltip>;
        } else {
            return null;
        }
    }));

export const PublicFavoriteStar = connect(mapStateToProps)(
    withStyles(styles)((props: { isPublicFavoriteVisible: boolean; className?: string; } & WithStyles<CssRules>) => {
        if (props.isPublicFavoriteVisible) {
            return <Tooltip enterDelay={500} title="Public Favorite"><PublicFavoriteIcon className={props.className || props.classes.icon} /></Tooltip>;
        } else {
            return null;
        }
    }));
