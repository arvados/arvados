// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Typography from '@material-ui/core/Typography';
import { WithStyles, withStyles, StyleRulesCallback } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import IconBase, { IconTypes } from '../icon/icon';

export interface EmptyStateDataProps {
    message: string;
    icon: IconTypes;
    details?: string;
}

type EmptyStateProps = EmptyStateDataProps & WithStyles<CssRules>;

class EmptyState extends React.Component<EmptyStateProps, {}> {

    render() {
        const { classes, message, details, icon, children } = this.props;
        return (
            <Typography className={classes.container} component="div">
                <IconBase icon={icon} className={classes.icon} />
                <Typography variant="body1" gutterBottom>{message}</Typography>
                { details && <Typography gutterBottom>{details}</Typography> }
                { children && <Typography gutterBottom>{children}</Typography> }
            </Typography>
        );
    }

}

type CssRules = 'container' | 'icon';
const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    container: {
        textAlign: 'center'
    },
    icon: {
        color: theme.palette.grey["500"],
        fontSize: '72px'
    }
});

export default withStyles(styles)(EmptyState);