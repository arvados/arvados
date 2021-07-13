// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Button, Grid, StyleRulesCallback, WithStyles, Typography, Tooltip } from '@material-ui/core';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';
import { withStyles } from '@material-ui/core';
import { IllegalNamingWarning } from '../warning/warning';

export interface Breadcrumb {
    label: string;
}

type CssRules = "item" | "currentItem" | "label";

const styles: StyleRulesCallback<CssRules> = theme => ({
    item: {
        opacity: 0.6
    },
    currentItem: {
        opacity: 1
    },
    label: {
        textTransform: "none"
    }
});

export interface BreadcrumbsProps {
    items: Breadcrumb[];
    onClick: (breadcrumb: Breadcrumb) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, breadcrumb: Breadcrumb) => void;
}

export const Breadcrumbs = withStyles(styles)(
    ({ classes, onClick, onContextMenu, items }: BreadcrumbsProps & WithStyles<CssRules>) =>
    <Grid container data-cy='breadcrumbs' alignItems="center" wrap="nowrap">
    {
        items.map((item, index) => {
            const isLastItem = index === items.length - 1;
            const isFirstItem = index === 0;
            return (
                <React.Fragment key={index}>
                    {isFirstItem ? null : <IllegalNamingWarning name={item.label} />}
                    <Tooltip title={item.label}>
                        <Button
                            data-cy={
                                isFirstItem
                                ? 'breadcrumb-first'
                                : isLastItem
                                    ? 'breadcrumb-last'
                                    : false}
                            color="inherit"
                            className={isLastItem ? classes.currentItem : classes.item}
                            onClick={() => onClick(item)}
                            onContextMenu={event => onContextMenu(event, item)}>
                            <Typography
                                noWrap
                                color="inherit"
                                className={classes.label}>
                                {item.label}
                            </Typography>
                        </Button>
                    </Tooltip>
                    {!isLastItem && <ChevronRightIcon color="inherit" className={classes.item} />}
                </React.Fragment>
            );
        })
    }
    </Grid>
);
