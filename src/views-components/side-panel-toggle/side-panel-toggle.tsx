// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Tooltip, IconButton  } from '@material-ui/core';
import { connect } from 'react-redux';
import { toggleSidePanel } from "store/side-panel/side-panel-action";
import { RootState } from 'store/store';

type collapseButtonProps = {
  isCollapsed: boolean;
  toggleSidePanel: (collapsedState: boolean) => void
}

export const COLLAPSE_ICON_SIZE = 20

const SidePanelToggle = (props: collapseButtonProps) =>{ 
const collapseButtonIconStyles = {
  root: {
    width: `${COLLAPSE_ICON_SIZE}px`,
    padding: '10px'
  },
  icon: {
  }
}


  return <Tooltip disableFocusListener title="Toggle Side Panel">
              <IconButton style={collapseButtonIconStyles.root} onClick={()=>{props.toggleSidePanel(props.isCollapsed)}}>
                <div style={collapseButtonIconStyles.icon}>{props.isCollapsed ? '>' : '<'}</div>
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
        toggleSidePanel: (collapsedState)=>{
            return dispatch(toggleSidePanel(collapsedState))
        }
    }
};

export default connect(mapStateToProps, mapDispatchToProps)(SidePanelToggle)