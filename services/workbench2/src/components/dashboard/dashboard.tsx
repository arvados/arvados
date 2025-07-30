// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { FavePinsSection } from './favorite-pins/favorite-pins-section';
import { RecentWorkflowRunsSection } from './recent-workflow-runs';
import { RecentlyVisitedSection } from './recently-visited';
import { setDashboardBreadcrumbs } from 'store/breadcrumbs/breadcrumbs-actions';

type CssRules = 'root' | 'section';

const styles: CustomStyleRulesCallback<CssRules> = () => ({
    root: {
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        padding: 0,
        paddingTop: '1rem',
    },
    section : {
        paddingBottom: '1rem'
    }
});

const mapDispatchToProps = (dispatch: Dispatch): DashboardProps => ({
    setDashboardBreadcrumbs: () => dispatch<any>(setDashboardBreadcrumbs()),
})

type DashboardProps = {
    setDashboardBreadcrumbs: () => void;
};


export const Dashboard = connect(null, mapDispatchToProps)(
    withStyles(styles)(({setDashboardBreadcrumbs, classes}: DashboardProps & WithStyles<CssRules>) => {

    useEffect(() => {
        setDashboardBreadcrumbs();
    }, [setDashboardBreadcrumbs]);

    return (
        <section className={classes.root}>
            <section className={classes.section}>
                <FavePinsSection />
            </section>
            <section className={classes.section}>
                <RecentlyVisitedSection />
            </section>
            <section className={classes.section}>
                <RecentWorkflowRunsSection />
            </section>
        </section>
    );
}));
