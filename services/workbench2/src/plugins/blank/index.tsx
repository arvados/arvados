// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Example plugin.

import { PluginConfig } from 'common/plugintypes';
import React from 'react';

export const register = (pluginConfig: PluginConfig) => {

    pluginConfig.centerPanelList.push((elms) => []);

    pluginConfig.sidePanelCategories.push((cats: string[]): string[] => []);

    pluginConfig.accountMenuList.push((elms) => []);
    pluginConfig.newButtonMenuList.push((elms) => []);

    pluginConfig.appBarLeft = <span />;
    pluginConfig.appBarMiddle = <span />;
    pluginConfig.appBarRight = <span />;
};
