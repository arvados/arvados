// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Typography from '@material-ui/core/Typography';
import { WithStyles, withStyles, StyleRulesCallback } from '@material-ui/core/styles';
import { ArvadosTheme } from 'src/common/custom-theme';
import { IconType } from '../icon/icon';

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

export interface EmptyStateDataProps {
    message: string;
    icon: IconType;
    details?: string;
}

type EmptyStateProps = EmptyStateDataProps & WithStyles<CssRules>;

export const EmptyState = withStyles(styles)(
    class extends React.Component<EmptyStateProps, {}> {
        render() {
            const {classes, message, details, icon: Icon, children} = this.props;
            return (
                <Typography className={classes.container} component="div">
                    <Icon className={classes.icon}/>
                    <Typography variant="body1" gutterBottom>{message}</Typography>
                    {details && <Typography gutterBottom>{details}</Typography>}
                    {children && <Typography gutterBottom>{children}</Typography>}
                </Typography>
            );
        }
    }
);
