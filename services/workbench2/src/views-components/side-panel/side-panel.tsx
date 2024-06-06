// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useRef, useEffect } from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { SidePanelTree, SidePanelTreeProps } from 'views-components/side-panel-tree/side-panel-tree';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { navigateFromSidePanel, setCurrentSideWidth } from 'store/side-panel/side-panel-action';
import { Grid } from '@material-ui/core';
import { SidePanelButton } from 'views-components/side-panel-button/side-panel-button';
import { RootState } from 'store/store';
import SidePanelToggle from 'views-components/side-panel-toggle/side-panel-toggle';
import { SidePanelCollapsed } from './side-panel-collapsed';

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
    },
    setCurrentSideWidth: width => {
        dispatch<any>(setCurrentSideWidth(width))
    }
});

const mapStateToProps = ({ router, sidePanel, detailsPanel }: RootState) => ({
    currentRoute: router.location ? router.location.pathname : '',
    isCollapsed: sidePanel.collapsedState,
    isDetailsPanelTransitioning: detailsPanel.isTransitioning
});

export const SidePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ classes, ...props }: WithStyles<CssRules> & SidePanelTreeProps & { currentRoute: string, isDetailsPanelTransitioning: boolean }) =>{

        const splitPaneRef = useRef<any>(null)

        useEffect(()=>{
            const splitPane = splitPaneRef?.current as Element

            if (!splitPane) return;

            const observerCallback: ResizeObserverCallback = (entries: ResizeObserverEntry[]) => {
                try {
                    //prevents potential infinite resize triggers
                    window.requestAnimationFrame((): void | undefined => {
                    if (!Array.isArray(entries) || !entries.length) {
                        return;
                    }
                        const width = entries[0].contentRect.width
                        props.setCurrentSideWidth(width)
                    });
                } catch (error) {
                    console.error('Error in resize observer callback', error);
                }
              };

            const observer = new ResizeObserver(observerCallback)

            observer.observe(splitPane)
            return ()=> observer.disconnect()
        }, [props])

            return (
                <Grid item xs>
                        {props.isCollapsed ? 
                            <div ref={splitPaneRef}>
                                <>
                                    <SidePanelToggle />
                                    <SidePanelCollapsed />
                                </>
                            </div>
                            :
                            <>
                                <div ref={splitPaneRef}>
                                    <Grid className={classes.topButtonContainer}>
                                        <SidePanelButton key={props.currentRoute} />
                                        <SidePanelToggle/>
                                    </Grid>
                                    <SidePanelTree {...props} />
                                </div>
                            </>
                        }
                    </Grid>
        )}
    ));
