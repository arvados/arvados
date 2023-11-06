// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { SidePanelTree, SidePanelTreeProps } from 'views-components/side-panel-tree/side-panel-tree';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { navigateFromSidePanel } from 'store/side-panel/side-panel-action';
import { Grid } from '@material-ui/core';
import { SidePanelButton } from 'views-components/side-panel-button/side-panel-button';
import { RootState } from 'store/store';
import SidePanelToggle from 'views-components/side-panel-toggle/side-panel-toggle';

const DRAWER_WIDTH = 240;

type CssRules = 'root' | 'topButtonContainer';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        background: theme.palette.background.paper,
        borderRight: `1px solid ${theme.palette.divider}`,
        height: '100%',
        overflowX: 'auto',
        width: DRAWER_WIDTH,
    },
    topButtonContainer: {
        display: 'flex',
        justifyContent: 'space-between'
    }
});

const mapDispatchToProps = (dispatch: Dispatch): SidePanelTreeProps => ({
    onItemActivation: id => {
        dispatch<any>(navigateFromSidePanel(id));
    }
});

const mapStateToProps = ({ router, sidePanel }: RootState) => ({
    currentRoute: router.location ? router.location.pathname : '',
    isCollapsed: sidePanel.collapsedState
});

export const SidePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ classes, ...props }: WithStyles<CssRules> & SidePanelTreeProps & { currentRoute: string }) =>
            <Grid item xs>
                {props.isCollapsed ? <SidePanelToggle /> :
                <>
                    <Grid className={classes.topButtonContainer}>
                        <SidePanelButton key={props.currentRoute} />
                        <SidePanelToggle/>
                    </Grid>
                    <SidePanelTree {...props} />
                </>}
            </Grid>
    ));
