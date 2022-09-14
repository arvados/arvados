// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { MutableRefObject, ReactElement, ReactNode, useEffect, useRef, useState } from 'react';
import {
    Button,
    Grid,
    Paper,
    StyleRulesCallback,
    Tooltip,
    withStyles,
    WithStyles
} from "@material-ui/core";
import { GridProps } from '@material-ui/core/Grid';
import { isArray } from 'lodash';
import { DefaultView } from 'components/default-view/default-view';
import { InfoIcon, InvisibleIcon, VisibleIcon } from 'components/icon/icon';
import { ReactNodeArray } from 'prop-types';
import classNames from 'classnames';

type CssRules = 'button' | 'buttonIcon' | 'content';

const styles: StyleRulesCallback<CssRules> = theme => ({
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
    },
});

interface MPVHideablePanelDataProps {
    name: string;
    visible: boolean;
    maximized: boolean;
    illuminated: boolean;
    children: ReactNode;
    panelRef?: MutableRefObject<any>;
}

interface MPVHideablePanelActionProps {
    doHidePanel: () => void;
    doMaximizePanel: () => void;
}

type MPVHideablePanelProps = MPVHideablePanelDataProps & MPVHideablePanelActionProps;

const MPVHideablePanel = ({doHidePanel, doMaximizePanel, name, visible, maximized, illuminated, ...props}: MPVHideablePanelProps) =>
    visible
    ? <>
        {React.cloneElement((props.children as ReactElement), { doHidePanel, doMaximizePanel, panelName: name, panelMaximized: maximized, panelIlluminated: illuminated, panelRef: props.panelRef })}
    </>
    : null;

interface MPVPanelDataProps {
    panelName?: string;
    panelMaximized?: boolean;
    panelIlluminated?: boolean;
    panelRef?: MutableRefObject<any>;
    forwardProps?: boolean;
    maxHeight?: string;
}

interface MPVPanelActionProps {
    doHidePanel?: () => void;
    doMaximizePanel?: () => void;
}

// Props received by panel implementors
export type MPVPanelProps = MPVPanelDataProps & MPVPanelActionProps;

type MPVPanelContentProps = {children: ReactElement} & MPVPanelProps & GridProps;

// Grid item compatible component for layout and MPV props passing
export const MPVPanelContent = ({doHidePanel, doMaximizePanel, panelName,
    panelMaximized, panelIlluminated, panelRef, forwardProps, maxHeight,
    ...props}: MPVPanelContentProps) => {
    useEffect(() => {
        if (panelRef && panelRef.current) {
            panelRef.current.scrollIntoView({behavior: 'smooth'});
        }
    }, [panelRef]);

    const mh = panelMaximized
        ? '100%'
        : maxHeight;

    return <Grid item style={{maxHeight: mh}} {...props}>
        <span ref={panelRef} /> {/* Element to scroll to when the panel is selected */}
        <Paper style={{height: '100%'}} elevation={panelIlluminated ? 8 : 0}>
            { forwardProps
                ? React.cloneElement(props.children, { doHidePanel, doMaximizePanel, panelName, panelMaximized })
                : props.children }
        </Paper>
    </Grid>;
}

export interface MPVPanelState {
    name: string;
    visible?: boolean;
}
interface MPVContainerDataProps {
    panelStates?: MPVPanelState[];
}
type MPVContainerProps = MPVContainerDataProps & GridProps;

// Grid container compatible component that also handles panel toggling.
const MPVContainerComponent = ({children, panelStates, classes, ...props}: MPVContainerProps & WithStyles<CssRules>) => {
    if (children === undefined || children === null || children === {}) {
        children = [];
    } else if (!isArray(children)) {
        children = [children];
    }
    const visibility = (children as ReactNodeArray).map((_, idx) =>
        !panelStates || // if panelStates wasn't passed, default to all visible panels
            (panelStates[idx] &&
                (panelStates[idx].visible || panelStates[idx].visible === undefined)));
    const [panelVisibility, setPanelVisibility] = useState<boolean[]>(visibility);
    const [brightenedPanel, setBrightenedPanel] = useState<number>(-1);
    const [selectedPanel, setSelectedPanel] = useState<number>(-1);
    const panelRef = useRef<any>(null);

    let panels: JSX.Element[] = [];
    let toggles: JSX.Element[] = [];

    if (isArray(children)) {
        for (let idx = 0; idx < children.length; idx++) {
            const showFn = (idx: number) => () => {
                setPanelVisibility([
                    ...panelVisibility.slice(0, idx),
                    true,
                    ...panelVisibility.slice(idx+1)
                ]);
                setSelectedPanel(idx);
            };
            const hideFn = (idx: number) => () => {
                setPanelVisibility([
                    ...panelVisibility.slice(0, idx),
                    false,
                    ...panelVisibility.slice(idx+1)
                ])
            };
            const maximizeFn = (idx: number) => () => {
                // Maximize X == hide all but X
                setPanelVisibility([
                    ...panelVisibility.slice(0, idx).map(() => false),
                    true,
                    ...panelVisibility.slice(idx+1).map(() => false),
                ])
            };
            const toggleIcon = panelVisibility[idx]
                ? <VisibleIcon className={classNames(classes.buttonIcon)} />
                : <InvisibleIcon className={classNames(classes.buttonIcon)}/>
            const panelName = panelStates === undefined
                ? `Panel ${idx+1}`
                : (panelStates[idx] && panelStates[idx].name) || `Panel ${idx+1}`;
            const btnVariant = "outlined";
            const btnTooltip = panelVisibility[idx]
                ? ''
                :`Show ${panelName} panel`;
            const panelIsMaximized = panelVisibility[idx] &&
                panelVisibility.filter(e => e).length === 1;

            toggles = [
                ...toggles,
                <Tooltip title={btnTooltip} disableFocusListener>
                    <Button variant={btnVariant} size="small" color="primary"
                        className={classNames(classes.button)}
                        onMouseEnter={() => {
                            setBrightenedPanel(idx);
                        }}
                        onMouseLeave={() => {
                            setBrightenedPanel(-1);
                        }}
                        onClick={showFn(idx)}>
                            {panelName}
                            {toggleIcon}
                    </Button>
                </Tooltip>
            ];

            const aPanel =
                <MPVHideablePanel key={idx} visible={panelVisibility[idx]} name={panelName}
                    panelRef={(idx === selectedPanel) ? panelRef : undefined}
                    maximized={panelIsMaximized} illuminated={idx === brightenedPanel}
                    doHidePanel={hideFn(idx)} doMaximizePanel={maximizeFn(idx)}>
                    {children[idx]}
                </MPVHideablePanel>;
            panels = [...panels, aPanel];
        };
    };

    return <Grid container {...props}>
        <Grid container item direction="row">
            { toggles.map((tgl, idx) => <Grid item key={idx}>{tgl}</Grid>) }
        </Grid>
        <Grid container item {...props} xs className={classes.content}>
            { panelVisibility.includes(true)
                ? panels
                : <Grid container item alignItems='center' justify='center'>
                    <DefaultView messages={["All panels are hidden.", "Click on the buttons above to show them."]} icon={InfoIcon} />
                </Grid> }
        </Grid>
    </Grid>;
};

export const MPVContainer = withStyles(styles)(MPVContainerComponent);