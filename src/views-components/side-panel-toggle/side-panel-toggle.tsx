// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Tooltip, IconButton } from '@material-ui/core';
import { connect } from 'react-redux';
import { toggleSidePanel } from "store/side-panel/side-panel-action";
import { RootState } from 'store/store';

type collapseButtonProps = {
    isCollapsed: boolean;
    toggleSidePanel: (collapsedState: boolean) => void
}

export const COLLAPSE_ICON_SIZE = 35

const SidePanelToggle = (props: collapseButtonProps) => {
    const collapseButtonIconStyles = {
        root: {
            width: `${COLLAPSE_ICON_SIZE}px`,
            height: `${COLLAPSE_ICON_SIZE}px`,
            marginTop: '0.4rem',
            marginLeft: '0.7rem',
        },
        icon: {
            opacity: '0.5',
        },
    }

    return <Tooltip disableFocusListener title="Toggle Side Panel">
        <IconButton data-cy="side-panel-toggle" style={collapseButtonIconStyles.root} onClick={() => { props.toggleSidePanel(props.isCollapsed) }}>
            <div>
                {props.isCollapsed ?
                    <img style={collapseButtonIconStyles.icon} src='/mui-start-icon.svg' alt='an arrow pointing right'/>
                    :
                    <img style={{ ...collapseButtonIconStyles.icon, transform: "rotate(180deg)"}} src='/mui-start-icon.svg' alt='an arrow pointing right'/>}
            </div>
        </IconButton>
    </Tooltip>
};

const mapStateToProps = (state: RootState) => {
    return {
        isCollapsed: state.sidePanel.collapsedState
    }
}

const mapDispatchToProps = (dispatch) => {
    return {
        toggleSidePanel: (collapsedState) => {
            return dispatch(toggleSidePanel(collapsedState))
        }
    }
};

export default connect(mapStateToProps, mapDispatchToProps)(SidePanelToggle)
