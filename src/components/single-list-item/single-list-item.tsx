// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import { ListItemIcon, ListItemText, Typography } from '@material-ui/core';
import { IconType } from '../icon/icon';
import * as classnames from "classnames";

export interface SingleListItemDataProps {
    icon: IconType;
    name: string;
    isActive?: boolean;
    hasMargin?: boolean;
}

type SingleListItemProps = SingleListItemDataProps & WithStyles<CssRules>;

class SingleListItem extends React.Component<SingleListItemProps, {}> {
    render() {
        const { classes, isActive, hasMargin, name, icon: Icon } = this.props;
        return (
            <Typography component='span' className={classes.root}>
                <ListItemIcon className={this.getListItemIconClassNames(hasMargin, isActive)}>
                    <Icon />
                </ListItemIcon>
                <ListItemText primary={
                    <Typography variant='body1' className={this.getListItemTextClassNames(isActive)}>
                        {name}
                    </Typography>
                } />
            </Typography>
        );
    }

    getListItemIconClassNames = (hasMargin?: boolean, isActive?: boolean) => {
        const { classes } = this.props;
        return classnames({
            [classes.hasMargin]: hasMargin,
            [classes.active]: isActive
        });
    }

    getListItemTextClassNames = (isActive?: boolean) => {
        const { classes } = this.props;
        return classnames(classes.listItemText, {
            [classes.active]: isActive
        });
    }


}
        
type CssRules = 'root' | 'listItemText' | 'hasMargin' | 'active';
        
const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        alignItems: 'center'
    },
    listItemText: {
        fontWeight: 700
    },
    active: {
        color: theme.palette.primary.main,
    },
    hasMargin: {
        marginLeft: '18px',
    },
});

export default withStyles(styles)(SingleListItem);