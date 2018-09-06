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

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        background: theme.palette.background.paper,
        borderRight: `1px solid ${theme.palette.divider}`,
        height: '100%',
        overflowX: 'auto',
        width: DRAWER_WITDH,
    }
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
    <div className={classes.root}>
        <SidePanelTree {...props} />
    </div>);
