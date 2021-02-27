// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch } from 'redux';
import { RootStore, RootState } from '~/store/store';
import { ResourcesState } from '~/store/resources/resources';
import { Location } from 'history';

export type ElementListReducer = (startingList: React.ReactElement[], itemClass?: string) => React.ReactElement[];
export type CategoriesListReducer = (startingList: string[]) => string[];
export type NavigateMatcher = (dispatch: Dispatch, getState: () => RootState, uuid: string) => boolean;
export type LocationChangeMatcher = (store: RootStore, pathname: string) => boolean;
export type EnableNew = (location: Location, currentItemId: string, currentUserUUID: string | undefined, resources: ResourcesState) => boolean;

export interface PluginConfig {
    // Customize the list of possible center panels by adding or removing Route components.
    centerPanelList: ElementListReducer[];

    // Customize the list of side panel categories
    sidePanelCategories: CategoriesListReducer[];

    // Add to the list of possible dialogs by adding dialog components.
    dialogs: React.ReactElement[];

    // Add navigation actions for identifiers
    navigateToHandlers: NavigateMatcher[];

    // Add handlers for navigation actions
    locationChangeHandlers: LocationChangeMatcher[];

    appBarLeft?: React.ReactElement;

    appBarMiddle?: React.ReactElement;

    appBarRight?: React.ReactElement;

    // Customize the list menu items in the account menu
    accountMenuList: ElementListReducer[];

    enableNewButtonMatchers: EnableNew[];

    newButtonMenuList: ElementListReducer[];
}
