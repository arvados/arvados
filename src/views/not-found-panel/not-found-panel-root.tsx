// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Location } from 'history';
import { StyleRulesCallback, WithStyles, withStyles, Paper, Grid } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { ClusterConfigJSON } from 'common/config';

export type CssRules = 'root' | 'title' | 'active';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        overflow: 'hidden',
        width: '100vw',
        height: '100vh'
    },
    title: {
        paddingLeft: theme.spacing.unit * 3,
        paddingTop: theme.spacing.unit * 3,
        paddingBottom: theme.spacing.unit * 3,
        fontSize: '18px'
    },
    active: {
        color: theme.customs.colors.green700,
        textDecoration: 'none',
    }
});

export interface NotFoundPanelOwnProps {
    notWrapped?: boolean;
}

export interface NotFoundPanelRootDataProps {
    location: Location<any> | null;
    clusterConfig: ClusterConfigJSON;
}

type NotFoundPanelRootProps = NotFoundPanelRootDataProps & NotFoundPanelOwnProps & WithStyles<CssRules>;

const getAdditionalMessage = (location: Location | null) => {
    if (!location) {
        return null;
    }

    const { pathname } = location;

    if (pathname.indexOf('collections') > -1) {
        const uuidHash = pathname.replace('/collections/', '');

        return (
            <p>
                Please make sure that provided UUID/ObjectHash '{uuidHash}' is valid.
            </p>
        );
    }

    return null;
};

const getEmailLink = (email: string, classes: Record<CssRules, string>) => {
    const { location: { href: windowHref } } = window;
    const href = `mailto:${email}?body=${encodeURIComponent('Problem while viewing page ')}${encodeURIComponent(windowHref)}&subject=${encodeURIComponent('Workbench problem report')}`;

    return (<a
        className={classes.active}
        href={href}>
        email us
    </a>);
};


export const NotFoundPanelRoot = withStyles(styles)(
    ({ classes, clusterConfig, location, notWrapped }: NotFoundPanelRootProps) => {

        const content = <Grid container justify="space-between" wrap="nowrap" alignItems="center">
            <div data-cy="not-found-content" className={classes.title}>
                <h2>Not Found</h2>
                {getAdditionalMessage(location)}
                <p>
                    The page you requested was not found,&nbsp;
                    {
                        !!clusterConfig.Mail && clusterConfig.Mail.SupportEmailAddress ?
                            getEmailLink(clusterConfig.Mail.SupportEmailAddress, classes) :
                            'email us'
                    }
                    &nbsp;if you suspect this is a bug.
                </p>
            </div>
        </Grid>;

        return !notWrapped ? <Paper data-cy="not-found-page"> {content}</Paper> : content;
    }
);
