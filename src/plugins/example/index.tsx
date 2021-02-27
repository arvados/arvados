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
import { DispatchProp, connect } from 'react-redux';
import { MenuItem } from "@material-ui/core";
import { propertiesActions } from '~/store/properties/properties-actions';
import { Location } from 'history';

const categoryName = "Plugin Example";
export const routePath = "/examplePlugin";
const propertyKey = "Example_menu_item_pressed_count";

interface ExampleProps {
    pressedCount: number;
}

const exampleMapStateToProps = (state: RootState) => ({ pressedCount: state.properties[propertyKey] || 0 });

const incrementPressedCount = (dispatch: Dispatch, pressedCount: number) => {
    dispatch(propertiesActions.SET_PROPERTY({ key: propertyKey, value: pressedCount + 1 }));
};

const ExampleMenuComponent = connect(exampleMapStateToProps)(
    ({ pressedCount, dispatch }: ExampleProps & DispatchProp<any>) =>
        <MenuItem onClick={() => incrementPressedCount(dispatch, pressedCount)}>Example menu item</MenuItem >
);

const ExamplePluginMainPanel = connect(exampleMapStateToProps)(
    ({ pressedCount }: ExampleProps) =>
        <Typography>
            This is a example main panel plugin.  The example menu item has been pressed {pressedCount} times.
	</Typography>);

export const register = (pluginConfig: PluginConfig) => {

    pluginConfig.centerPanelList.push((elms) => {
        elms.push(<Route path={routePath} component={ExamplePluginMainPanel} />);
        return elms;
    });

    pluginConfig.accountMenuList.push((elms) => {
        elms.push(<ExampleMenuComponent />);
        return elms;
    });

    pluginConfig.newButtonMenuList.push((elms) => {
        elms.push(<ExampleMenuComponent />);
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

    pluginConfig.enableNewButtonMatchers.push((location: Location) => (!!matchPath(location.pathname, { path: routePath, exact: true })));
};
