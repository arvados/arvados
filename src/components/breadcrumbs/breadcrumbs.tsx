// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Button, Grid, StyleRulesCallback, WithStyles } from '@material-ui/core';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';
import { withStyles } from '@material-ui/core';

export interface Breadcrumb {
    label: string;
}

interface BreadcrumbsProps {
    items: Breadcrumb[];
    onClick: (breadcrumb: Breadcrumb) => any;
}

const Breadcrumbs: React.SFC<BreadcrumbsProps & WithStyles<CssRules>> = (props) => {
    const { classes, onClick, items } = props;
    return <Grid container alignItems="center">
        {
            getInactiveItems(items).map((item, index) => (
                <React.Fragment key={index}>
                    <Button
                        color="inherit"
                        className={classes.inactiveItem}
                        onClick={() => onClick(item)}
                    >
                        {item.label}
                    </Button>
                    <ChevronRightIcon color="inherit" className={classes.inactiveItem} />
                </React.Fragment>
            ))
        }
        {
            getActiveItem(items).map((item, index) => (
                <Button
                    color="inherit"
                    key={index}
                    onClick={() => onClick(item)}
                >
                    {item.label}
                </Button>
            ))
        }
    </Grid>;
};

const getInactiveItems = (items: Breadcrumb[]) => {
    return items.slice(0, -1);
};

const getActiveItem = (items: Breadcrumb[]) => {
    return items.slice(-1);
};

type CssRules = 'inactiveItem';

const styles: StyleRulesCallback<CssRules> = theme => {
    const { unit } = theme.spacing;
    return {
        inactiveItem: {
            opacity: 0.6
        }
    };
};

export default withStyles(styles)(Breadcrumbs);

