// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { MutableRefObject, ReactElement, ReactNode, useEffect, useRef, useState } from 'react';
import { connect } from 'react-redux';
import { RouterState } from "react-router-redux";
import { RootState } from 'store/store';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Paper, Tabs, Tab } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { GridProps } from '@mui/material/Grid';
import { isArray, isEqual } from 'lodash';
import classNames from 'classnames';

type CssRules =
    | 'exclusiveGridContainerRoot'
    | 'symmetricTabs'
    | 'gridItemRoot'
    | 'paperRoot'
    | 'button'
    | 'exclusiveContentPaper'
    | 'exclusiveContent'
    | 'tab'
    | 'selectedTab';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    exclusiveGridContainerRoot: {
        marginTop: 0,
    },
    symmetricTabs: {
        "& button": {
            flexBasis: "0",
        },
    },
    gridItemRoot: {
        paddingTop: '0 !important',
        width: '100%',
    },
    paperRoot: {
        height: '100%',
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
    },
    button: {
        padding: '2px 5px',
        marginRight: '5px',
    },
    exclusiveContent: {
        overflow: 'auto',
        margin: 0,
    },
    exclusiveContentPaper: {
        boxShadow: 'none',
    },
    tab: {
        flexGrow: 1,
        flexShrink: 1,
        maxWidth: 'initial',
        minWidth: 'fit-content',
        padding: '0 5px',
        borderBottom: `1px solid ${theme.palette.grey[300]}`,
    },
    selectedTab: {
    },
});

interface MPVHideablePanelDataProps {
    name: string;
    visible: boolean;
    children: ReactNode;
    panelRef?: MutableRefObject<any>;
    paperClassName?: string;
}

const MPVHideablePanel = ({ name, visible, paperClassName, ...props }: MPVHideablePanelDataProps) =>
    visible
        ? <>
            {React.cloneElement((props.children as ReactElement), {
                panelName: name,
                panelRef: props.panelRef,
                paperClassName,
            })}
        </>
        : null;

interface MPVPanelDataProps {
    panelName?: string;
    panelRef?: MutableRefObject<any>;
    forwardProps?: boolean;
    maxHeight?: string;
    minHeight?: string;
    paperClassName?: string;
}

// Props received by panel implementors
export type MPVPanelProps = MPVPanelDataProps;

type MPVPanelContentProps = { children: ReactElement } & MPVPanelProps & GridProps;

// Grid item compatible component for layout and MPV props passing
export const MPVPanelContent = React.memo(({ panelName,
    panelRef, forwardProps, maxHeight, minHeight, paperClassName,
    ...props }: MPVPanelContentProps) => {

    return <Grid item style={{ maxHeight: maxHeight, minHeight, padding: '4px' }} {...props}>
        <span ref={panelRef} /> {/* Element to scroll to when the panel is selected */}
        <Paper style={{ height: '100%' }} elevation={0}>
            {forwardProps
                ? React.cloneElement(props.children, { panelName, paperClassName })
                : React.cloneElement(props.children)}
        </Paper>
    </Grid>;
}, preventRerender);

// return true to prevent re-render, false to allow re-render
function preventRerender(prevProps: MPVPanelContentProps, nextProps: MPVPanelContentProps) {
    if (!isEqual(prevProps.children, nextProps.children)) {
        return false;
    }
    return true;
}

export interface MPVPanelState {
    name: string;
    visible?: boolean;
}
interface MPVContainerDataProps {
    panelStates?: MPVPanelState[];
    router: RouterState;
}
type MPVContainerProps = MPVContainerDataProps & GridProps;

const mapStateToProps = (state: RootState): Pick<MPVContainerDataProps, 'router'> => ({
    router: state.router,
});

// Grid container compatible component that also handles panel toggling.
const MPVContainerComponent = ({ children, panelStates, classes, router, ...props }: MPVContainerProps & WithStyles<CssRules>) => {
    if (children === undefined || children === null || Object.keys(children).length === 0) {
        children = [];
    } else if (!isArray(children)) {
        children = [children];
    } else {
        children = children.filter(child => child !== null);
    }

    const [initialVisibility, setInitialVisibility] = useState<boolean[]>(getInitialVisibility(panelStates, children as []));

    useEffect(() => {
        setInitialVisibility(getInitialVisibility(panelStates, children as []));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [(children as []).length]);

    const [panelVisibility, setPanelVisibility] = useState<boolean[]>(initialVisibility);
    const currentSelectedPanel = panelVisibility.findIndex(Boolean);
    const [selectedPanel, setSelectedPanel] = useState<number>(-1);
    const panelRef = useRef<any>(null);

    // Reset MPV to initial state when route changes
    const currentRoute = router.location ? router.location.pathname : "";
    useEffect(() => {
        setPanelVisibility(initialVisibility);
        setSelectedPanel(initialVisibility.indexOf(true));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currentRoute, initialVisibility]);

    let panels: JSX.Element[] = [];
    let tabs: JSX.Element[] = [];
    let tabBar: JSX.Element = <></>;

    if (isArray(children)) {
        const showFn = (idx: number) => () => {
            // Hide all other panels
            setPanelVisibility(Array.from({ length: (children as []).length }, (_, index) => index === idx));
            setSelectedPanel(idx);
        };

        for (let idx = 0; idx < children.length; idx++) {
            const panelName = panelStates === undefined
                ? `Panel ${idx + 1}`
                : (panelStates[idx] && panelStates[idx].name) || `Panel ${idx + 1}`;

            tabs = [
                ...tabs,
                <>{panelName}</>
            ];

            const aPanel =
                <MPVHideablePanel
                    key={idx}
                    visible={panelVisibility[idx]}
                    name={panelName}
                    paperClassName={classes.exclusiveContentPaper}
                    panelRef={(idx === selectedPanel) ? panelRef : undefined}
                    >
                    {children[idx]}
                </MPVHideablePanel>;
            panels = [...panels, aPanel];
        };

        tabBar = (
            <Tabs className={classes.symmetricTabs} value={currentSelectedPanel} onChange={(e, val) => showFn(val)()} data-cy={"mpv-tabs"}>
                {tabs.map((tgl, idx) => <Tab className={classNames(classes.tab, idx === selectedPanel ? classes.selectedTab : '')} key={idx} label={tgl} />)}
            </Tabs>);
    };

    const content = <Grid container direction="column" item {...props} xs className={classes.exclusiveContent}>
                        {panelVisibility.includes(true) && panels}
                    </Grid>;

        return (
            <Grid container {...props} className={classNames(classes.exclusiveGridContainerRoot, props.className)}>
                <Grid item {...props} className={classes.gridItemRoot}>
                    <Paper className={classes.paperRoot}>
                        {tabBar}
                        {content}
                    </Paper>
                </Grid>
            </Grid>);
};

const getInitialVisibility = (panelStates: MPVPanelState[] | undefined, children: ReactNode[]) => {
    if (panelStates && panelStates.some(state => state.visible)) {
        return panelStates.map((panelState) => panelState.visible || false);
    }
    // if panelStates wasn't passed or passed with none selected, default to first panel visible
    return new Array(children.length).fill(false).map((_, idx) => idx === 0);
}

export const MPVContainer = connect(mapStateToProps)(withStyles(styles)(MPVContainerComponent));