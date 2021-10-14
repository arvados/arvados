// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement, ReactNode, useState } from 'react';
import { Button, Grid, StyleRulesCallback, Tooltip, withStyles, WithStyles } from "@material-ui/core";
import { GridProps } from '@material-ui/core/Grid';
import { isArray } from 'lodash';
import { DefaultView } from 'components/default-view/default-view';
import { InfoIcon, InvisibleIcon, VisibleIcon } from 'components/icon/icon';
import { ReactNodeArray } from 'prop-types';
import classNames from 'classnames';

type CssRules = 'button' | 'buttonIcon';

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
});

interface MPVHideablePanelDataProps {
    name: string;
    visible: boolean;
    children: ReactNode;
}

interface MPVHideablePanelActionProps {
    doHidePanel: () => void;
}

type MPVPanelProps = MPVHideablePanelDataProps & MPVHideablePanelActionProps;

const MPVHideablePanel = ({doHidePanel, name, visible, ...props}: MPVPanelProps) =>
    visible
    ? <>
        {React.cloneElement((props.children as ReactElement), { doHidePanel, panelName: name })}
    </>
    : null;

interface MPVPanelContentDataProps {
    panelName?: string;
    children: ReactElement;
}

interface MPVPanelContentActionProps {
    doHidePanel?: () => void;
}

type MPVPanelContentProps = MPVPanelContentDataProps & MPVPanelContentActionProps & GridProps;

// Grid item compatible component for layout and MPV props passing
export const MPVPanelContent = ({doHidePanel, panelName, ...props}: MPVPanelContentProps) =>
    <Grid item {...props}>
        {React.cloneElement(props.children, { doHidePanel, panelName })}
    </Grid>;

export interface MPVContainerDataProps {
    panelNames?: string[];
}

type MPVContainerProps = MPVContainerDataProps & GridProps;

// Grid container compatible component that also handles panel toggling.
const MPVContainerComponent = ({children, panelNames, classes, ...props}: MPVContainerProps & WithStyles<CssRules>) => {
    if (children === undefined || children === null || children === {}) {
        children = [];
    } else if (!isArray(children)) {
        children = [children];
    }
    const visibility = (children as ReactNodeArray).map(() => true);
    const [panelVisibility, setPanelVisibility] = useState<boolean[]>(visibility);

    let panels: JSX.Element[] = [];
    let toggles: JSX.Element[] = [];

    if (isArray(children)) {
        for (let idx = 0; idx < children.length; idx++) {
            const toggleFn = (idx: number) => () => {
                setPanelVisibility([
                    ...panelVisibility.slice(0, idx),
                    !panelVisibility[idx],
                    ...panelVisibility.slice(idx+1)
                ])
            };
            const toggleIcon = panelVisibility[idx]
                ? <VisibleIcon className={classNames(classes.buttonIcon)} />
                : <InvisibleIcon className={classNames(classes.buttonIcon)}/>
            const panelName = panelNames === undefined
                ? `Panel ${idx+1}`
                : panelNames[idx] || `Panel ${idx+1}`;
            const toggleVariant = panelVisibility[idx]
                ? "raised"
                : "flat";
            const toggleTooltip = panelVisibility[idx]
                ? `Hide ${panelName} panel`
                : `Show ${panelName} panel`;

            toggles = [
                ...toggles,
                <Tooltip title={toggleTooltip} disableFocusListener>
                    <Button variant={toggleVariant} size="small" color="primary"
                        className={classNames(classes.button)}
                        onClick={toggleFn(idx)}>
                            {panelName}
                            {toggleIcon}
                    </Button>
                </Tooltip>
            ];

            const aPanel =
                <MPVHideablePanel visible={panelVisibility[idx]} name={panelName} doHidePanel={toggleFn(idx)}>
                    {children[idx]}
                </MPVHideablePanel>;
            panels = [...panels, aPanel];
        };
    };

    return <Grid container {...props}>
        <Grid item>
            { toggles }
        </Grid>
        { panelVisibility.includes(true)
            ? panels
            : <Grid container alignItems='center' justify='center'>
                <DefaultView messages={["All panels are hidden.", "Click on the buttons above to show them."]} icon={InfoIcon} />
            </Grid> }
    </Grid>;
};

export const MPVContainer = withStyles(styles)(MPVContainerComponent);