// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { MutableRefObject, ReactElement, ReactNode, useEffect, useRef, useState } from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Button, Grid, Paper, Tooltip, Tabs, Tab } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { GridProps } from '@mui/material/Grid';
import { isArray } from 'lodash';
import { DefaultView } from 'components/default-view/default-view';
import { InfoIcon } from 'components/icon/icon';
import classNames from 'classnames';

type CssRules =
    | 'root'
    | 'buttonRow'
    | 'gridContainerRoot'
    | 'exclusiveGridContainerRoot'
    | 'gridItemRoot'
    | 'paperRoot'
    | 'button'
    | 'buttonIcon'
    | 'content'
    | 'exclusiveContentPaper'
    | 'tabs';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    root: {
        marginTop: '0',
    },
    buttonRow: {
        marginBottom: '15px',
    },
    gridContainerRoot: {
        marginTop: '10px',
    },
    exclusiveGridContainerRoot: {
        marginTop: 0,
    },
    gridItemRoot: {
        paddingTop: '0 !important',
    },
    paperRoot: {
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
    },
    button: {
        padding: '2px 5px',
        marginRight: '5px',
    },
    buttonIcon: {
        boxShadow: 'none',
        padding: '2px 0px 2px 5px',
        fontSize: '1rem'
    },
    content: {
        overflow: 'auto',
        maxWidth: 'initial',
    },
    exclusiveContentPaper: {
        boxShadow: 'none',
    },
    tabs: {
        flexGrow: 1,
        flexShrink: 1,
        maxWidth: 'initial',
        borderBottom: `1px solid ${theme.palette.grey[300]}`,
    },
});

interface MPVHideablePanelDataProps {
    name: string;
    visible: boolean;
    maximized: boolean;
    illuminated: boolean;
    children: ReactNode;
    panelRef?: MutableRefObject<any>;
    paperClassName?: string;
}

interface MPVHideablePanelActionProps {
    doHidePanel: () => void;
    doMaximizePanel: () => void;
    doUnMaximizePanel: () => void;
}

type MPVHideablePanelProps = MPVHideablePanelDataProps & MPVHideablePanelActionProps;

const MPVHideablePanel = ({ doHidePanel, doMaximizePanel, doUnMaximizePanel, name, visible, maximized, illuminated, paperClassName, ...props }: MPVHideablePanelProps) =>
    visible
        ? <>
            {React.cloneElement((props.children as ReactElement), {
                doHidePanel,
                doMaximizePanel,
                doUnMaximizePanel,
                panelName: name,
                panelMaximized: maximized,
                panelIlluminated: illuminated,
                panelRef: props.panelRef,
                paperClassName,
            })}
        </>
        : null;

interface MPVPanelDataProps {
    panelName?: string;
    panelMaximized?: boolean;
    panelIlluminated?: boolean;
    panelRef?: MutableRefObject<any>;
    forwardProps?: boolean;
    maxHeight?: string;
    minHeight?: string;
    paperClassName?: string;
}

interface MPVPanelActionProps {
    doHidePanel?: () => void;
    doMaximizePanel?: () => void;
    doUnMaximizePanel?: () => void;
}

// Props received by panel implementors
export type MPVPanelProps = MPVPanelDataProps & MPVPanelActionProps;

type MPVPanelContentProps = { children: ReactElement } & MPVPanelProps & GridProps;

// Grid item compatible component for layout and MPV props passing
export const MPVPanelContent = ({ doHidePanel, doMaximizePanel, doUnMaximizePanel, panelName,
    panelMaximized, panelIlluminated, panelRef, forwardProps, maxHeight, minHeight, paperClassName,
    ...props }: MPVPanelContentProps) => {
    useEffect(() => {
        if (panelRef && panelRef.current) {
            panelRef.current.scrollIntoView({ alignToTop: true });
        }
    }, [panelRef]);

    const maxH = panelMaximized
        ? '100%'
        : maxHeight;

    return <Grid item style={{ maxHeight: maxH, minHeight }} {...props}>
        <span ref={panelRef} /> {/* Element to scroll to when the panel is selected */}
        <Paper style={{ height: '100%' }} elevation={panelIlluminated ? 8 : 0}>
            {forwardProps
                ? React.cloneElement(props.children, { doHidePanel, doMaximizePanel, doUnMaximizePanel, panelName, panelMaximized, paperClassName })
                : React.cloneElement(props.children, { paperClassName })}
        </Paper>
    </Grid>;
}

export interface MPVPanelState {
    name: string;
    visible?: boolean;
}
interface MPVContainerDataProps {
    panelStates?: MPVPanelState[];
    mutuallyExclusive?: boolean;
}
type MPVContainerProps = MPVContainerDataProps & GridProps;

// Grid container compatible component that also handles panel toggling.
const MPVContainerComponent = ({ children, panelStates, classes, ...props }: MPVContainerProps & WithStyles<CssRules>) => {
    if (children === undefined || children === null || Object.keys(children).length === 0) {
        children = [];
    } else if (!isArray(children)) {
        children = [children];
    }
    const initialVisibility = (children as ReactNode[]).map((_, idx) =>
        !panelStates || // if panelStates wasn't passed, default to all visible panels
        (panelStates[idx] &&
            (panelStates[idx].visible || panelStates[idx].visible === undefined)));
    const [panelVisibility, setPanelVisibility] = useState<boolean[]>(initialVisibility);
    const [previousPanelVisibility, setPreviousPanelVisibility] = useState<boolean[]>(initialVisibility);
    const [highlightedPanel, setHighlightedPanel] = useState<number>(-1);
    const currentSelectedPanel = panelVisibility.findIndex(Boolean);
    const [selectedPanel, setSelectedPanel] = useState<number>(-1);
    const panelRef = useRef<any>(null);

    let panels: JSX.Element[] = [];
    let buttons: JSX.Element[] = [];
    let tabs: JSX.Element[] = [];
    let buttonBar: JSX.Element = <></>;

    if (isArray(children)) {
        const showFn = (idx: number) => () => {
            setPreviousPanelVisibility(initialVisibility);
            if (props.mutuallyExclusive) {
                // Hide all other panels
                setPanelVisibility([
                    ...(new Array(idx).fill(false)),
                    true,
                    ...(new Array(panelVisibility.length-(idx+1)).fill(false)),
                ]);
            } else {
                setPanelVisibility([
                    ...panelVisibility.slice(0, idx),
                    true,
                    ...panelVisibility.slice(idx + 1)
                ]);
            }
            setSelectedPanel(idx);
        };
        const hideFn = (idx: number) => () => {
            setPreviousPanelVisibility(initialVisibility);
            setPanelVisibility([
                ...panelVisibility.slice(0, idx),
                false,
                ...panelVisibility.slice(idx+1)
            ])
        };
        const maximizeFn = (idx: number) => () => {
            setPreviousPanelVisibility(panelVisibility);
            // Maximize X == hide all but X
            setPanelVisibility([
                ...panelVisibility.slice(0, idx).map(() => false),
                true,
                ...panelVisibility.slice(idx+1).map(() => false),
            ]);
        };
        const unMaximizeFn = (idx: number) => () => {
            setPanelVisibility(previousPanelVisibility);
            setSelectedPanel(idx);
        }
        for (let idx = 0; idx < children.length; idx++) {
            const panelName = panelStates === undefined
                ? `Panel ${idx + 1}`
                : (panelStates[idx] && panelStates[idx].name) || `Panel ${idx + 1}`;
            const btnVariant = panelVisibility[idx]
                ? "contained"
                : "outlined";
            const btnTooltip = panelVisibility[idx]
                ? ``
                : `Open ${panelName} panel`;
            const panelIsMaximized = panelVisibility[idx] &&
                panelVisibility.filter(e => e).length === 1;

            buttons = [
                ...buttons,
                <Tooltip title={btnTooltip} disableFocusListener>
                    <Button variant={btnVariant} size="small" color="primary"
                        className={classNames(classes.button)}
                        onMouseEnter={() => {
                            setHighlightedPanel(idx);
                        }}
                        onMouseLeave={() => {
                            setHighlightedPanel(-1);
                        }}
                        onClick={showFn(idx)}>
                        {panelName}
                    </Button>
                </Tooltip>
            ];

            tabs = [
                ...tabs,
                <>{panelName}</>
            ];

            const aPanel =
                <MPVHideablePanel
                    key={idx}
                    visible={panelVisibility[idx]}
                    name={panelName}
                    paperClassName={props.mutuallyExclusive ? classes.exclusiveContentPaper : undefined}
                    panelRef={(idx === selectedPanel) ? panelRef : undefined}
                    maximized={panelIsMaximized} illuminated={idx === highlightedPanel}
                    doHidePanel={hideFn(idx)} doMaximizePanel={maximizeFn(idx)} doUnMaximizePanel={panelIsMaximized ? unMaximizeFn(idx) : () => null}>
                    {children[idx]}
                </MPVHideablePanel>;
            panels = [...panels, aPanel];
        };

        buttonBar = props.mutuallyExclusive ?
            <Tabs value={currentSelectedPanel} onChange={(e, val) => showFn(val)()} data-cy={"mpv-tabs"}>
                {tabs.map((tgl, idx) => <Tab className={classes.tabs} key={idx} label={tgl} />)}
            </Tabs> :
            <Grid container item direction="row">
                {buttons.map((tgl, idx) => <Grid item key={idx}>{tgl}</Grid>)}
            </Grid>;
    };

    const content = <Grid container item {...props} xs className={classes.content}
        onScroll={() => setSelectedPanel(-1)}>
        {panelVisibility.includes(true)
            ? panels
            : <Grid container item alignItems='center' justifyContent='center'>
                <DefaultView messages={["All panels are hidden.", "Click on the buttons above to show them."]} icon={InfoIcon} />
            </Grid>}
    </Grid>;

    if (props.mutuallyExclusive) {
        return <Grid container {...props} className={classNames(classes.exclusiveGridContainerRoot, props.className)}>
            <Grid item {...props} className={classes.gridItemRoot}>
                <Paper className={classes.paperRoot}>
                    {buttonBar}
                    {content}
                </Paper>
            </Grid>
        </Grid>;
    } else {
        return <Grid container {...props} className={classNames(classes.gridContainerRoot, props.className)}>
            {buttonBar}
            {content}
        </Grid>;
    }
};

export const MPVContainer = withStyles(styles)(MPVContainerComponent);
