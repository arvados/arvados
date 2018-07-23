// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DefaultIcon, IconType, ProjectsIcon } from '../../components/icon/icon';
import { EmptyResource } from '../../models/empty';
import { DetailsData } from "./details-data";
import Typography from "@material-ui/core/Typography";
import { StyleRulesCallback, WithStyles, withStyles } from "@material-ui/core/styles";
import { ArvadosTheme } from "../../common/custom-theme";
import Icon from "@material-ui/core/Icon/Icon";

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
    children?: React.ReactNode;
}

type EmptyStateProps = EmptyStateDataProps & WithStyles<CssRules>;

const EmptyState = withStyles(styles)(
    ({ classes, details, message, children, icon: Icon }: EmptyStateProps) =>
        <Typography className={classes.container} component="div">
            <Icon className={classes.icon}/>
            <Typography variant="body1" gutterBottom>{message}</Typography>
            {details && <Typography gutterBottom>{details}</Typography>}
            {children && <Typography gutterBottom>{children}</Typography>}
        </Typography>
);

export class EmptyDetails extends DetailsData<EmptyResource> {
    getIcon(className?: string) {
        return <ProjectsIcon className={className}/>;
    }

    getDetails() {
       return <EmptyState icon={DefaultIcon} message='Select a file or folder to view its details.'/>;
    }
}
