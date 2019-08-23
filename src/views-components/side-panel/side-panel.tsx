// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { SidePanelTree, SidePanelTreeProps } from '~/views-components/side-panel-tree/side-panel-tree';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { navigateFromSidePanel } from '~/store/side-panel/side-panel-action';
import { Grid } from '@material-ui/core';
import { SidePanelButton } from '~/views-components/side-panel-button/side-panel-button';
import { RootState } from '~/store/store';

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

const mapStateToProps = ({ router }: RootState) => ({
    currentRoute: router.location ? router.location.pathname : '',
});

export const SidePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ classes, ...props }: WithStyles<CssRules> & SidePanelTreeProps & { currentRoute: string }) =>
            <Grid item xs>
                <SidePanelButton key={props.currentRoute} />
                <SidePanelTree {...props} />
            </Grid>
    ));
