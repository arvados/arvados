// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ContextMenuAction } from '../context-menu-action-set';
import { Divider as DividerComponent, StyleRulesCallback, withStyles } from '@material-ui/core';
import { WithStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { VerticalLineDivider } from 'components/icon/icon';

type CssRules = 'horizontal' | 'vertical';

const styles:StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
  horizontal: {
      backgroundColor: 'black',
  },
  vertical: {
    color: theme.palette.grey["400"],
    margin: 'auto 0',
    transform: 'scaleY(1.25)',
  },
});

export const VerticalLine = withStyles(styles)((props: WithStyles<CssRules>) => {
  return  <VerticalLineDivider className={props.classes.vertical} />;
});

export const HorizontalLine = withStyles(styles)((props: WithStyles<CssRules>) => {
  return  <DividerComponent variant='middle' className={props.classes.horizontal} />;
});

export const horizontalMenuDivider: ContextMenuAction = {
  name: 'Divider',
  icon: () => null,
  component: VerticalLine,
  execute: () => null,
};

export const verticalMenuDivider: ContextMenuAction = {
  name: 'Divider',
  icon: () => null,
  component: HorizontalLine,
  execute: () => null,
};