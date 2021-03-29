// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch, Middleware } from 'redux';
import { RootStore, RootState } from '~/store/store';
import { ResourcesState } from '~/store/resources/resources';
import { Location } from 'history';
import { ServiceRepository } from "~/services/services";

export type ElementListReducer = (startingList: React.ReactElement[], itemClass?: string) => React.ReactElement[];
export type CategoriesListReducer = (startingList: string[]) => string[];
export type NavigateMatcher = (dispatch: Dispatch, getState: () => RootState, uuid: string) => boolean;
export type LocationChangeMatcher = (store: RootStore, pathname: string) => boolean;
export type EnableNew = (location: Location, currentItemId: string, currentUserUUID: string | undefined, resources: ResourcesState) => boolean;
export type MiddlewareListReducer = (startingList: Middleware[], services: ServiceRepository) => Middleware[];

/* Workbench Plugin API

   Code to your plugin should go into a subdirectory of '~/plugins'.

   Your plugin should implement a "register" function, which will be
   called with an object with the PluginConfig interface described
   below.  The register function may make in-place modifications to
   the pluginConfig object, but to preserve composability, it is
   strongly advised this should be limited to push()ing new values
   onto the various lists of hooks.

   To enable a plugin, edit 'plugins.tsx', import the register
   function exported by the plugin, and add a call to the register
   function following the examples in the comments.  Then, build a new
   Workbench package that includes the plugin.

   Be aware that because plugins heavily leverage workbench, and in
   fact must be compiled together, they are considered "derived works"
   and so _must_ be license-compatible with AGPL-3.0.

 */

export interface PluginConfig {

    /* During initialization, each
     * function in the callback list will be called with the list of
     * react - router "Route" components that will be used select what should
     * be displayed in the central panel based on the navigation bar.
     *
     * The callback function may add, edit, or remove items from this list,
     * and return a new list of components, which will be passed to the next
     * function in `centerPanelList`.
     *
     * The hooks are applied in `views/workbench/workbench.tsx`.
     *  */
    centerPanelList: ElementListReducer[];

    /* During initialization, each
     * function in the callback list will be called with the list of strings
     * that are the top-level categories in the left hand navigation tree.
     *
     * The callback function may add, edit, or remove items from this list,
     * and return a new list of strings, which will be passed to the next
     * function in `sidePanelCategories`.
     *
     * The hooks are applied in `store/side-panel-tree/side-panel-tree-actions.ts`.
     *  */
    sidePanelCategories: CategoriesListReducer[];

    /* This is a list of additional dialog box components.
     * Dialogs are components that are wrapped using the "withDialog()" method.
     *
     * These are added to the list in `views/workbench/workbench.tsx`.
     *  */
    dialogs: React.ReactElement[];

    /* This is a list of additional navigation matchers.
     * These are callbacks that are called by the navigateTo(uuid) method to
     * set the path in the navigation bar to display the desired resource.
     * Each handler should return "true" if the uuid was handled and "false or "undefined" if not.
     *
     * These are used in `store/navigation/navigation-action.tsx`.
     *  */
    navigateToHandlers: NavigateMatcher[];

    /* This is a list of additional location change matchers.
     * These are callbacks called when the URL in the navigation bar changes
     * (this could be in response to "navigateTo()" or due to the user
     * entering/changing the URL directly).
     *
     * The Route components in centerPanelList should
     * automatically change in response to navigation.  The
     * purpose of these handlers is trigger additional loading,
     * such as fetching the object contents that will be
     * displayed.
     *
     * Each handler should return "true" if the path was handled and "false or "undefined" if not.
     *
     * These are used in `routes/route-change-handlers.ts`.
     */
    locationChangeHandlers: LocationChangeMatcher[];

    /* Replace the left side of the app bar.  Normally, this displays
     * the site banner.
     *
     * Note: unlike most of the other hooks, this is not composable.
     * This completely replaces that section of the app bar.  Multiple
     * plugins setting this value will conflict.
     *
     * Used in 'views-components/main-app-bar/main-app-bar.tsx'
     */
    appBarLeft?: React.ReactElement;

    /* Replace the middle part of the app bar.  Normally, this displays
     * the search bar.
     *
     * Note: unlike most of the other hooks, this is not composable.
     * This completely replaces that section of the app bar.  Multiple
     * plugins setting this value will conflict.
     *
     * Used in 'views-components/main-app-bar/main-app-bar.tsx'
     */
    appBarMiddle?: React.ReactElement;

    /* Replace the right part of the app bar.  Normally, this displays
     * the admin menu and help menu.
     * (Note: the user menu can be customized separately using accountMenuList)
     *
     * Note: unlike most of the other hooks, this is not composable.
     * This completely replaces that section of the app bar.  Multiple
     * plugins setting this value will conflict.
     *
     * Used in 'views-components/main-app-bar/main-app-bar.tsx'
     */
    appBarRight?: React.ReactElement;

    /* During initialization, each
     * function in the callback list will be called with the menu items that
     * will appear in the "user account" menu.
     *
     * The callback function may add, edit, or remove items from this list,
     * and return a new list of menu items, which will be passed to the next
     * function in `accountMenuList`.
     *
     * The hooks are applied in 'views-components/main-app-bar/account-menu.tsx'.
     *  */
    accountMenuList: ElementListReducer[];

    /* Each function in this list is called to determine if the the "NEW" button
     * should be enabled or disabled.  If any function returns "true", the button
     * (and corresponding drop-down menu) will be enabled.
     *
     * The hooks are applied in 'views-components/side-panel-button/side-panel-button.tsx'.
     *  */
    enableNewButtonMatchers: EnableNew[];

    /* During initialization, each
     * function in the callback list will be called with the menu items that
     * will appear in the "NEW" dropdown menu.
     *
     * The callback function may add, edit, or remove items from this list,
     * and return a new list of menu items, which will be passed to the next
     * function in `newButtonMenuList`.
     *
     * The hooks are applied in 'views-components/side-panel-button/side-panel-button.tsx'.
     *  */
    newButtonMenuList: ElementListReducer[];

    /* Add Middlewares to the Redux store.
     *
     * Middlewares intercept redux actions before they get to the reducer, and
     * may produce side effects.  For example, the REQUEST_ITEMS action is intercepted by a middleware to
     * trigger a load of data table contents.
     *
     * https://redux.js.org/tutorials/fundamentals/part-4-store#middleware
     *
     * Used in 'store/store.ts'
     *  */
    middlewares: MiddlewareListReducer[];
}
