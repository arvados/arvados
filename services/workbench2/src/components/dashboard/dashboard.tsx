// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { FavePinsSection } from './favorite-pins/favorite-pins-section';
import { RecentWorkflowRunsSection } from './recent-workflow-runs';
import { RecentlyVisitedSection } from './recently-visited';

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


export const Dashboard = withStyles(styles)(({classes}: WithStyles<CssRules>) => {
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
});
