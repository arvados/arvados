// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ContextMenuAction } from '../context-menu-action-set';
import { Divider as DividerComponent, StyleRulesCallback, withStyles } from '@material-ui/core';
import { WithStyles } from '@material-ui/core/styles';

type CssRules = 'root';

const styles:StyleRulesCallback<CssRules> = () => ({
  root: {
      backgroundColor: 'black',
  },
});

type DividerProps = {
  orthogonality: 'vertical' | 'horizontal';
};

export const Divider = withStyles(styles)((props: DividerProps & WithStyles<CssRules>) => {
  return  <DividerComponent variant='middle' className={props.classes.root} />;
});

export const menuDivider: ContextMenuAction = {
  name: 'divider',
  component: Divider,
  execute: () => null,
};