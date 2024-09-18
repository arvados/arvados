// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { WarningIcon } from "components/icon/icon";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { DialogContentText } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'container' | 'text';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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