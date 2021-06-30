// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Example workbench plugin.  The entry point is the "register" method.

import { PluginConfig } from 'common/plugintypes';
import React from 'react';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { push } from "react-router-redux";
import { Route, matchPath } from "react-router";
import { RootStore } from 'store/store';
import { activateSidePanelTreeItem } from 'store/side-panel-tree/side-panel-tree-actions';
import { setSidePanelBreadcrumbs } from 'store/breadcrumbs/breadcrumbs-actions';
import { Location } from 'history';
import { handleFirstTimeLoad } from 'store/workbench/workbench-actions';
import {
    ExampleDialog,
    ExamplePluginMainPanel,
    ExampleMenuComponent,
    ExampleDialogMenuComponent
} from './exampleComponents';

const categoryName = "Plugin Example";
export const routePath = "/examplePlugin";

export const register = (pluginConfig: PluginConfig) => {

    // Add this component to the main panel.  When the app navigates
    // to '/examplePlugin' it will render ExamplePluginMainPanel.
    pluginConfig.centerPanelList.push((elms) => {
        elms.push(<Route path={routePath} component={ExamplePluginMainPanel} />);
        return elms;
    });

    // Add ExampleDialogMenuComponent to the upper-right user account menu
    pluginConfig.accountMenuList.push((elms, menuItemClass) => {
        elms.push(<ExampleDialogMenuComponent className={menuItemClass} />);
        return elms;
    });

    // Add ExampleMenuComponent to the "New" button dropdown.
    pluginConfig.newButtonMenuList.push((elms, menuItemClass) => {
        elms.push(<ExampleMenuComponent className={menuItemClass} />);
        return elms;
    });

    // Add a hook so that when the 'Plugin Example' entry in the left
    // hand tree view is clicked, which calls navigateTo('Plugin Example'),
    // it will be implemented by navigating to '/examplePlugin'
    pluginConfig.navigateToHandlers.push((dispatch: Dispatch, getState: () => RootState, uuid: string) => {
        if (uuid === categoryName) {
            dispatch(push(routePath));
            return true;
        }
        return false;
    });

    // Adds 'Plugin Example' to the left hand tree view.
    pluginConfig.sidePanelCategories.push((cats: string[]): string[] => { cats.push(categoryName); return cats; });

    // When the location changes to '/examplePlugin', make sure
    // 'Plugin Example' in the left hand tree view is selected, and
    // make sure the breadcrumbs are updated.
    pluginConfig.locationChangeHandlers.push((store: RootStore, pathname: string): boolean => {
        if (matchPath(pathname, { path: routePath, exact: true })) {
            store.dispatch(handleFirstTimeLoad(
                (dispatch: Dispatch) => {
                    dispatch<any>(activateSidePanelTreeItem(categoryName));
                    dispatch<any>(setSidePanelBreadcrumbs(categoryName));
                }));
            return true;
        }
        return false;
    });

    // The "New" button can enabled or disabled based on the current
    // context or selection.  This adds a new callback to that will
    // enable the "New" button when the location is '/examplePlugin'
    pluginConfig.enableNewButtonMatchers.push((location: Location) => (!!matchPath(location.pathname, { path: routePath, exact: true })));

    // Add the example dialog box to the list of dialog box controls.
    pluginConfig.dialogs.push(<ExampleDialog />);
};
