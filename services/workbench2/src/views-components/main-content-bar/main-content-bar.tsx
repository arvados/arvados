// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";

import { Toolbar, StyleRulesCallback, Grid, WithStyles, withStyles } from "@material-ui/core";
import { Breadcrumbs } from "views-components/breadcrumbs/breadcrumbs";
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import RefreshButton from "components/refresh-button/refresh-button";
import { loadSidePanelTreeProjects } from "store/side-panel-tree/side-panel-tree-actions";
import { Dispatch } from "redux";

type CssRules = 'mainBar' | 'breadcrumbContainer' | 'infoTooltip';

const styles: StyleRulesCallback<CssRules> = theme => ({
    mainBar: {
        flexWrap: 'nowrap',
    },
    breadcrumbContainer: {
        overflow: 'hidden',
    },
    infoTooltip: {
        marginTop: '-10px',
        marginLeft: '10px',
    }
});

interface MainContentBarProps {
    onRefreshPage: () => void;
    onDetailsPanelToggle: () => void;
}

const mapStateToProps = (state: RootState) => ({
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
