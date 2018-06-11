// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Button, Grid, StyleRulesCallback, WithStyles } from '@material-ui/core';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';
import { withStyles } from '@material-ui/core';

interface Breadcrumb {
    label: string
}

interface BreadcrumbsDataProps {
    items: Breadcrumb[]
}

interface BreadcrumbsActionProps {
    onClick: (breadcrumb: Breadcrumb) => any
}

type BreadcrumbsProps = BreadcrumbsDataProps & BreadcrumbsActionProps & WithStyles<CssRules>;

class Breadcrumbs extends React.Component<BreadcrumbsProps> {

    render() {
        const { classes, onClick } = this.props;
        return <Grid container alignItems="center">
            {
                this.getInactiveItems().map((item, index) => (
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
                this.getActiveItem().map((item, index) => (
                    <Button
                        color="inherit"
                        key={index}
                        onClick={() => onClick(item)}
                    >
                        {item.label}
                    </Button>
                ))
            }
        </Grid>
    }

    getInactiveItems = () => {
        return this.props.items.slice(0, -1)
    }

    getActiveItem = () => {
        return this.props.items.slice(-1)
    }

}

type CssRules = 'inactiveItem'

const styles: StyleRulesCallback<CssRules> = theme => {
    const { unit } = theme.spacing
    return {
        inactiveItem: {
            opacity: 0.6
        }
    }
}

export default withStyles(styles)(Breadcrumbs)

