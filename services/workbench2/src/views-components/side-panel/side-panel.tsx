// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useRef, useEffect } from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { SidePanelTree, SidePanelTreeProps } from 'views-components/side-panel-tree/side-panel-tree';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { navigateFromSidePanel, setCurrentSideWidth } from 'store/side-panel/side-panel-action';
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
    setCurrentSideWidth: width => {
        dispatch<any>(setCurrentSideWidth(width))
    }
});

const mapStateToProps = ({ router, sidePanel }: RootState) => ({
    currentRoute: router.location ? router.location.pathname : '',
    isCollapsed: sidePanel.collapsedState,
    currentSideWidth: sidePanel.currentSideWidth,
});

export const SidePanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ classes, ...props }: WithStyles<CssRules> & SidePanelTreeProps ) =>{

        const splitPaneRef = useRef<any>(null)

        useEffect(()=>{
            const splitPane = splitPaneRef?.current as Element

            if (!splitPane) return;

            const observerCallback: ResizeObserverCallback = (entries: ResizeObserverEntry[]) => {
                //entries[0] targets the left side of the split pane
                const width = entries[0].contentRect.width
                if (width === props.currentSideWidth) return;

                //prevents potential infinite resize triggers
                window.requestAnimationFrame((): void | undefined => {
                  if (!Array.isArray(entries) || !entries.length) {
                      props.setCurrentSideWidth(width)
                    return;
                  }
                });
              };

            const observer = new ResizeObserver(observerCallback)

            observer.observe(splitPane)
            return ()=> observer.disconnect()
        }, [props])

            return (
                <Grid item xs className={classes.sidePanelGridItem}>
                    {props.isCollapsed ?
                        <div ref={splitPaneRef}>
                            <div>
                                <SidePanelToggle />
                                <SidePanelCollapsed />
                            </div>
                        </div>
                            :
                        <div ref={splitPaneRef}>
                            <div className={classes.topButtonContainer}>
                                <SidePanelButton key={props.currentRoute} />
                                <SidePanelToggle/>
                            </div>
                            <SidePanelTree {...props} />
                        </div>
                    }
                </Grid>
        )}
    ));
