// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";

import { Toolbar, StyleRulesCallback, IconButton, Tooltip, Grid, WithStyles, withStyles } from "@material-ui/core";
import { DetailsIcon } from "components/icon/icon";
import { Breadcrumbs } from "views-components/breadcrumbs/breadcrumbs";
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import * as Routes from 'routes/routes';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import RefreshButton from "components/refresh-button/refresh-button";
import { loadSidePanelTreeProjects } from "store/side-panel-tree/side-panel-tree-actions";
import { Dispatch } from "redux";

type CssRules = 'mainBar' | 'infoTooltip';

const styles: StyleRulesCallback<CssRules> = theme => ({
    mainBar: {
        flexWrap: 'nowrap',
    },
    infoTooltip: {
        marginTop: '-10px',
        marginLeft: '10px',
    }
});

interface MainContentBarProps {
    onRefreshPage: () => void;
    onDetailsPanelToggle: () => void;
    buttonVisible: boolean;
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

const mapStateToProps = (state: RootState) => ({
    buttonVisible: isButtonVisible(state),
    projectUuid: state.detailsPanel.resourceUuid,
});

const mapDispatchToProps = () => (dispatch: Dispatch) => ({
    onDetailsPanelToggle: () => dispatch<any>(toggleDetailsPanel()),
    onRefreshButtonClick: (id) => {
        dispatch<any>(loadSidePanelTreeProjects(id));
    }
});

export const MainContentBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    (props: MainContentBarProps & WithStyles<CssRules> & any) =>
        <Toolbar><Grid container className={props.classes.mainBar}>
            <Grid container item xs alignItems="center">
                <Breadcrumbs />
            </Grid>
            <Grid item>
                <RefreshButton onClick={() => {
                    props.onRefreshButtonClick(props.projectUuid);
                }} />
            </Grid>
            <Grid item>
                {props.buttonVisible && <Tooltip title="Additional Info">
                    <IconButton data-cy="additional-info-icon"
                        color="inherit"
                        className={props.classes.infoTooltip}
                        onClick={props.onDetailsPanelToggle}>
                        <DetailsIcon />
                    </IconButton>
                </Tooltip>}
            </Grid>
        </Grid></Toolbar>
));
