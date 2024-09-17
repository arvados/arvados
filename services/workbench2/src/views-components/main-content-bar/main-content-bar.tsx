// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Toolbar, IconButton, Tooltip, Grid } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DetailsIcon } from "components/icon/icon";
import { Breadcrumbs } from "views-components/breadcrumbs/breadcrumbs";
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import * as Routes from 'routes/routes';
import RefreshButton from "components/refresh-button/refresh-button";
import { loadSidePanelTreeProjects } from "store/side-panel-tree/side-panel-tree-actions";
import { Dispatch } from "redux";

type CssRules = 'mainBar' | 'breadcrumbContainer';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    mainBar: {
        flexWrap: 'nowrap',
    },
    breadcrumbContainer: {
        overflow: 'hidden',
    },
});

interface MainContentBarProps {
    onRefreshPage: () => void;
    buttonVisible: boolean;
    projectUuid: string;
}

const isButtonVisible = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    return Routes.matchCollectionsContentAddressRoute(pathname) ||
        Routes.matchPublicFavoritesRoute(pathname) ||
        Routes.matchGroupDetailsRoute(pathname) ||
        Routes.matchGroupsRoute(pathname) ||
        Routes.matchUsersRoute(pathname) ||
        Routes.matchSearchResultsRoute(pathname) ||
        Routes.matchSharedWithMeRoute(pathname) ||
        Routes.matchProcessRoute(pathname) ||
        Routes.matchCollectionRoute(pathname) ||
        Routes.matchProjectRoute(pathname) ||
        Routes.matchAllProcessesRoute(pathname) ||
        Routes.matchTrashRoute(pathname) ||
        Routes.matchFavoritesRoute(pathname);
};

const mapStateToProps = (state: RootState) => {
    const currentRoute = state.router.location?.pathname.split('/') || [];
    const projectUuid = currentRoute[currentRoute.length - 1];

    return {
        buttonVisible: isButtonVisible(state),
        projectUuid,
    }
};

const mapDispatchToProps = () => (dispatch: Dispatch) => ({
    onRefreshButtonClick: (id) => {
        dispatch<any>(loadSidePanelTreeProjects(id));
    }
});

export const MainContentBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    (props: MainContentBarProps & WithStyles<CssRules> & any) =>
        <Toolbar><Grid container className={props.classes.mainBar}>
            <Grid container item xs alignItems="center" className={props.classes.breadcrumbContainer}>
                <Breadcrumbs />
            </Grid>
            <Grid item>
                <RefreshButton onClick={() => {
                    props.onRefreshButtonClick(props.projectUuid);
                }} />
            </Grid>
        </Grid></Toolbar>
));
