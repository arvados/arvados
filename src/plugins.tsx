// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PluginConfig } from '~/common/plugintypes';

export const pluginConfig: PluginConfig = {
    centerPanelList: [],
    sidePanelCategories: [],
    dialogs: [],
    navigateToHandlers: [],
    locationChangeHandlers: [],
    appBarLeft: undefined,
    appBarMiddle: undefined,
    appBarRight: undefined,
    accountMenuList: [],
    enableNewButtonMatchers: [],
    newButtonMenuList: [],
    middlewares: []
};

// Starting here, import and register your Workbench 2 plugins. //

// import { register as blankUIPluginRegister } from '~/plugins/blank/index';
// import { register as sampleTrackerPluginRegister } from '~/plugins/sample-tracker/index';
// import { studyListRoutePath } from '~/plugins/sample-tracker/studyList';
// import { register as examplePluginRegister, routePath as exampleRoutePath } from '~/plugins/example/index';
// import { register as rootRedirectRegister } from '~/plugins/root-redirect/index';

// blankUIPluginRegister(pluginConfig);
// sampleTrackerPluginRegister(pluginConfig);
// rootRedirectRegister(pluginConfig, studyListRoutePath);
