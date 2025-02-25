// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { SidePanelTree, SidePanelTreeProps } from 'views-components/side-panel-tree/side-panel-tree';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { navigateFromSidePanel } from 'store/side-panel/side-panel-action';
import { Grid } from '@mui/material';
import { SidePanelButton } from 'views-components/side-panel-button/side-panel-button';
import { RootState } from 'store/store';
import SidePanelToggle from 'views-components/side-panel-toggle/side-panel-toggle';
import { SidePanelCollapsed } from './side-panel-collapsed';

type CssRules = 'sidePanelGridItem' | 'topButtonContainer';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    sidePanelGridItem: {
        maxWidth: 'inherit',
        wordBreak: 'break-word',
    },
    topButtonContainer: {
        display: 'flex',
        justifyContent: 'space-between'
    }
});

const mapDispatchToProps = (dispatch: Dispatch): SidePanelTreeProps => ({
    onItemActivation: id => {
        dispatch<any>(navigateFromSidePanel(id));
    },
});

const mapStateToProps = ({ router, sidePanel }: RootState): Partial<SidePanelTreeProps> => ({
    currentRoute: router.location ? router.location.pathname : '',
    isCollapsed: sidePanel.collapsedState,
});

export const SidePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ classes, ...props }: WithStyles<CssRules> & SidePanelTreeProps ) => (
                <Grid item xs className={classes.sidePanelGridItem}>
                    {props.isCollapsed ?
                        <div>
                            <SidePanelToggle />
                            <SidePanelCollapsed />
                        </div>
                            :
                        <div>
                            <div className={classes.topButtonContainer}>
                                <SidePanelButton key={props.currentRoute} />
                                <SidePanelToggle/>
                            </div>
                            <SidePanelTree {...props} />
                        </div>
                    }
                </Grid>
        )
    )
);
