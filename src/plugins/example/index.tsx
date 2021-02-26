// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Example plugin.

import { PluginConfig } from '~/common/plugintypes';
import * as React from 'react';
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { push } from "react-router-redux";
import { Typography } from "@material-ui/core";
import { Route, matchPath } from "react-router";
import { RootStore } from '~/store/store';
import { activateSidePanelTreeItem } from '~/store/side-panel-tree/side-panel-tree-actions';
import { setSidePanelBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';

const categoryName = "Plugin Example";
export const routePath = "/examplePlugin";

const ExamplePluginMainPanel = (props: {}) => {
    return <Typography>
        This is a example main panel plugin.
    </Typography>;
};

export const register = (pluginConfig: PluginConfig) => {

    pluginConfig.centerPanelList.push((elms) => {
        elms.push(<Route path={routePath} component={ExamplePluginMainPanel} />);
        return elms;
    });

    pluginConfig.navigateToHandlers.push((dispatch: Dispatch, getState: () => RootState, uuid: string) => {
        if (uuid === categoryName) {
            dispatch(push(routePath));
            return true;
        }
        return false;
    });

    pluginConfig.sidePanelCategories.push((cats: string[]): string[] => { cats.push(categoryName); return cats; });

    pluginConfig.locationChangeHandlers.push((store: RootStore, pathname: string): boolean => {
        if (matchPath(pathname, { path: routePath, exact: true })) {
            store.dispatch(activateSidePanelTreeItem(categoryName));
            store.dispatch<any>(setSidePanelBreadcrumbs(categoryName));
            return true;
        }
        return false;
    });
};
