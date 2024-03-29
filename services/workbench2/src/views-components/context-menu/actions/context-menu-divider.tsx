// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ContextMenuAction } from '../context-menu-action-set';
import { Divider as DividerComponent, StyleRulesCallback, withStyles } from '@material-ui/core';
import { WithStyles } from '@material-ui/core/styles';
import { VerticalLineDivider } from 'components/icon/icon';

type CssRules = 'horizontal' | 'vertical';

const styles:StyleRulesCallback<CssRules> = () => ({
  horizontal: {
      backgroundColor: 'black',
  },
  vertical: {
      backgroundColor: 'black',
      transform: 'rotate(90deg)',
  },
});

export const VerticalLine = withStyles(styles)((props: WithStyles<CssRules>) => {
  return  <DividerComponent variant='middle' className={props.classes.vertical}/>;
});

export const HorizontalLine = withStyles(styles)((props: WithStyles<CssRules>) => {
  return  <DividerComponent variant='middle' className={props.classes.horizontal} />;
});

//msToolbar only renders icon and not component
export const horizontalMenuDivider: ContextMenuAction = {
  name: 'divider',
  icon: VerticalLineDivider,
  component: VerticalLine,
  execute: () => null,
};

export const verticalMenuDivider: ContextMenuAction = {
  name: 'divider',
  icon: () => null,
  component: HorizontalLine,
  execute: () => null,
};