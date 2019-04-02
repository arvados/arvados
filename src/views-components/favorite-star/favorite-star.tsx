// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { FavoriteIcon } from "~/components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "~/store/store";
import { withStyles, StyleRulesCallback, WithStyles } from "@material-ui/core";

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
    withStyles(styles)((props: { isFavoriteVisible: boolean; isPublicFavoriteVisible: boolean; className?: string; } & WithStyles<CssRules>) => {
        if (props.isPublicFavoriteVisible) {
            return <FavoriteIcon className={props.className || props.classes.icon} />;
        } else if (props.isFavoriteVisible) {
            return <FavoriteIcon className={props.className || props.classes.icon} />;
        } else {
            return null;
        }
    }));
