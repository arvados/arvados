// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, LinearProgress } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { User } from "models/user";
import { ArvadosTheme } from 'common/custom-theme';
import { WorkbenchPanel } from 'views/workbench/workbench';
import { LoginPanel } from 'views/login-panel/login-panel';
import { InactivePanel } from 'views/inactive-panel/inactive-panel';
import { WorkbenchLoadingScreen } from 'views/workbench/workbench-loading-screen';
import { MainAppBar } from 'views-components/main-app-bar/main-app-bar';
import { Routes, matchLinkAccountRoute } from 'routes/routes';
import { RouterState } from "react-router-redux";
import parse from 'parse-duration';
import { Config } from 'common/config';
import { LinkAccountPanelState, LinkAccountPanelStatus } from 'store/link-account-panel/link-account-panel-reducer';
import { WORKBENCH_LOADING_SCREEN } from 'store/progress-indicator/progress-indicator-actions';

type CssRules = 'root';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        overflow: 'hidden',
        width: '100vw',
        height: '100vh'
    }
});

export interface MainPanelRootDataProps {
    user?: User;
    progressIndicator: string[];
    buildInfo: string;
    uuidPrefix: string;
    linkAccountPanel: LinkAccountPanelState;
    config: Config;
    sidePanelIsCollapsed: boolean;
    isTransitioning: boolean;
    currentSideWidth: number;
    router: RouterState;
}

interface MainPanelRootDispatchProps {
    toggleSidePanel: () => void,
    setCurrentRouteUuid: (uuid: string | null) => void;
}

type MainPanelRootProps = MainPanelRootDataProps & MainPanelRootDispatchProps & WithStyles<CssRules>;

export const MainPanelRoot = withStyles(styles)(
    ({ classes, progressIndicator, user, buildInfo, uuidPrefix, config, linkAccountPanel,
        sidePanelIsCollapsed, isTransitioning, currentSideWidth, setCurrentRouteUuid, router}: MainPanelRootProps) =>{

            const working = progressIndicator.length > 0;
            const loading = progressIndicator.includes(WORKBENCH_LOADING_SCREEN);
            const isLinkingPath = router.location ? matchLinkAccountRoute(router.location.pathname) !== null : false;
            const currentRoute = router.location ? router.location.pathname : '';
            const isNotLinking = linkAccountPanel.status === LinkAccountPanelStatus.NONE || linkAccountPanel.status === LinkAccountPanelStatus.INITIAL;
            const siteBanner = config.clusterConfig.Workbench.SiteName;
            const sessionIdleTimeout = parse(config.clusterConfig.Workbench.IdleTimeout, 's') || 0;

            useEffect(() => {
                const splitRoute = currentRoute.split('/');
                const uuid = splitRoute[splitRoute.length - 1];
                if(Object.values(Routes).includes(`/${uuid}`) === false) {
                    setCurrentRouteUuid(uuid);
                } else {
                    setCurrentRouteUuid(null);
                }
                // eslint-disable-next-line react-hooks/exhaustive-deps
            }, [currentRoute]);

        return loading
            ? <WorkbenchLoadingScreen />
            : <>
            {isNotLinking && <MainAppBar
                user={user}
                buildInfo={buildInfo}
                uuidPrefix={uuidPrefix}
                siteBanner={siteBanner}
                >
                {working
                    ? <LinearProgress color="secondary" data-cy="linear-progress" />
                    : null}
            </MainAppBar>}
            <Grid container direction="column" className={classes.root}>
                {user
                    ? (user.isActive || (!user.isActive && isLinkingPath)
                    ? <WorkbenchPanel
                        isNotLinking={isNotLinking}
                        isUserActive={user.isActive}
                        sessionIdleTimeout={sessionIdleTimeout}
                        sidePanelIsCollapsed={sidePanelIsCollapsed}
                        isTransitioning={isTransitioning}
                        currentSideWidth={currentSideWidth}/>
                    : <InactivePanel />)
                    : <LoginPanel />}
            </Grid>
        </>
    }
);
