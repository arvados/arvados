// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PluginConfig } from 'common/plugintypes';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { SidePanelTreeCategory } from 'store/side-panel-tree/side-panel-tree-actions';
import { push } from "react-router-redux";

export const register = (pluginConfig: PluginConfig, redirect: string) => {

    pluginConfig.navigateToHandlers.push((dispatch: Dispatch, getState: () => RootState, uuid: string) => {
        if (uuid === SidePanelTreeCategory.PROJECTS) {
            dispatch(push(redirect));
            return true;
        }
        return false;
    });
};
