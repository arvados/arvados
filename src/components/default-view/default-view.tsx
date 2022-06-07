// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import { Typography } from '@material-ui/core';
import { IconType } from '../icon/icon';
import classnames from "classnames";

type CssRules = 'root' | 'icon' | 'message';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        textAlign: 'center'
    },
    icon: {
        color: theme.palette.grey["500"],
        fontSize: '4.5rem'
    },
    message: {
        color: theme.palette.grey["500"]
    }
});

export interface DefaultViewDataProps {
    classRoot?: string;
    messages: string[];
    filtersApplied?: boolean;
    classMessage?: string;
    icon: IconType;
    classIcon?: string;
}

type DefaultViewProps = DefaultViewDataProps & WithStyles<CssRules>;

export const DefaultView = withStyles(styles)(
    ({ classes, classRoot, messages, classMessage, icon: Icon, classIcon }: DefaultViewProps) =>
        <Typography className={classnames([classes.root, classRoot])} component="div">
            <Icon className={classnames([classes.icon, classIcon])} />
            {messages.map((msg: string, index: number) => {
                return <Typography key={index}
                    className={classnames([classes.message, classMessage])}>{msg}</Typography>;
            })}
        </Typography>
);
