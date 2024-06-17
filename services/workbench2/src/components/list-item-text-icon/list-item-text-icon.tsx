// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { ListItemIcon, ListItemText, Typography } from '@mui/material';
import { IconType } from '../icon/icon';
import classnames from "classnames";

type CssRules = 'root' | 'listItemText' | 'hasMargin' | 'active';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        alignItems: 'center'
    },
    listItemText: {
        fontWeight: 400
    },
    active: {
        color: theme.palette.primary.main,
    },
    hasMargin: {
        marginLeft: theme.spacing(1),
    }
});

export interface ListItemTextIconDataProps {
    icon: IconType;
    name: string;
    isActive?: boolean;
    hasMargin?: boolean;
    iconSize?: number;
    nameDecorator?: JSX.Element;
}

type ListItemTextIconProps = ListItemTextIconDataProps & WithStyles<CssRules>;

export const ListItemTextIcon = withStyles(styles)(
    class extends React.Component<ListItemTextIconProps, {}> {
        render() {
            const { classes, isActive, hasMargin, name, icon: Icon, iconSize, nameDecorator } = this.props;
            return (
                <Typography component='span' className={classes.root}>
                    <ListItemIcon className={classnames({
                            [classes.hasMargin]: hasMargin,
                            [classes.active]: isActive
                        })}>

                        <Icon style={{ fontSize: `${iconSize}rem` }} />
                    </ListItemIcon>
                    {nameDecorator || null}
                    <ListItemText primary={
                        <Typography className={classnames(classes.listItemText, {
                                [classes.active]: isActive
                            })}>
                            {name}
                        </Typography>
                    } />
                </Typography>
            );
        }
    }
);
