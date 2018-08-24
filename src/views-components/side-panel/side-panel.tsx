// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { ArvadosTheme } from '~/common/custom-theme';
import { SidePanelTree, SidePanelTreeProps } from '~/views-components/side-panel-tree/side-panel-tree';
import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';
import { navigateFromSidePanel } from '../../store/side-panel/side-panel-action';

const DRAWER_WITDH = 240;

type CssRules = 'drawerPaper' | 'toolbar';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    drawerPaper: {
        position: 'relative',
        width: DRAWER_WITDH,
        display: 'flex',
        flexDirection: 'column',
        paddingTop: 58,
        overflow: 'auto',
    },
    toolbar: theme.mixins.toolbar
});

const mapDispatchToProps = (dispatch: Dispatch): SidePanelTreeProps => ({
    onItemActivation: id => {
        dispatch<any>(navigateFromSidePanel(id));
    }
});

export const SidePanel = compose(
    withStyles(styles),
    connect(undefined, mapDispatchToProps)
)(({ classes, ...props }: WithStyles<CssRules> & SidePanelTreeProps) =>
    <Drawer
        variant="permanent"
        classes={{ paper: classes.drawerPaper }}>
        <div className={classes.toolbar} />
        <SidePanelTree {...props} />
    </Drawer>);
