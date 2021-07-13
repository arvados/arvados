// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { WarningIcon } from "components/icon/icon";
import { StyleRulesCallback, DialogContentText, WithStyles, withStyles } from "@material-ui/core";
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'container' | 'text';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    container: {
        display: 'flex',
        alignItems: 'center',
    },
    text: {
        paddingLeft: '8px'
    }
});

interface WarningCollectionProps {
    text: string;
}

export const WarningCollection = withStyles(styles)(({ classes, text }: WarningCollectionProps & WithStyles<CssRules>) =>
    <span className={classes.container}>
        <WarningIcon />
        <DialogContentText className={classes.text}>{text}</DialogContentText>
    </span>);