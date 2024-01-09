// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useMemo } from "react";
import MoreVertIcon from "@material-ui/icons/MoreVert";
import classnames from "classnames";
import { IconButton, Menu, MenuItem, StyleRulesCallback, WithStyles, withStyles } from "@material-ui/core";
import { ArvadosTheme } from "common/custom-theme";

type CssRules = 'inOverflowMenu'

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
  inOverflowMenu: {
    "&:hover": {
      backgroundColor: "transparent"
    }
  }
});

export const OverflowMenu = withStyles(styles)((props: any & WithStyles<CssRules>) => {
  const { children, className, visibilityMap, classes } = props
  const [anchorEl, setAnchorEl] = useState(null);
  const open = Boolean(anchorEl);
  const handleClick = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const shouldShowMenu = useMemo(
    () => Object.values(visibilityMap).some((v) => v === false),
    [visibilityMap]
  );
  if (!shouldShowMenu) {
    return null;
  }
  return (
    <div className={className}>
      <IconButton
        aria-label="more"
        aria-controls="long-menu"
        aria-haspopup="true"
        onClick={handleClick}
      >
        <MoreVertIcon />
      </IconButton>
      <Menu
        id="long-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleClose}
      >
        {React.Children.map(children, (child:any) => {
          if (!visibilityMap[child.props["data-targetid"]]) {
            return (
              <MenuItem key={child} onClick={handleClose}>
                {React.cloneElement(child, {
                  className: classnames(child.className, classes.inOverflowMenu)
                })}
              </MenuItem>
            );
          }
          return null;
        })}
      </Menu>
    </div>
  );
})