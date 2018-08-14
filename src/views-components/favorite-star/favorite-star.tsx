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
    visible: state.favorites[props.resourceUuid],
});

export const FavoriteStar = connect(mapStateToProps)(
    withStyles(styles)((props: { visible: boolean; className?: string; } & WithStyles<CssRules>) =>
        props.visible ? <FavoriteIcon className={props.className || props.classes.icon} /> : null
    ));
