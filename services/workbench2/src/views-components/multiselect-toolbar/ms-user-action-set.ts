// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AdvancedIcon, AttributesIcon } from "components/icon/icon";
import { MultiSelectMenuAction, MultiSelectMenuActionSet, MultiSelectMenuActionNames } from "./ms-menu-actions";
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { dialogActions } from "store/dialog/dialog-actions";
import { UserResource } from "models/user";
import { getResource } from "store/resources/resources";

const { ATTRIBUTES, API_DETAILS } = MultiSelectMenuActionNames

export const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';

const openUserAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_ATTRIBUTES_DIALOG, data }));
    };

const msUserAttributes: MultiSelectMenuAction  = {
  name: ATTRIBUTES,
  icon: AttributesIcon,
  hasAlts: false,
  isForMulti: false,
  execute: (dispatch, resources) => {
      dispatch<any>(openUserAttributes(resources[0].uuid));
  },
};

const msAdvancedAction: MultiSelectMenuAction  = {
  name: API_DETAILS,
  icon: AdvancedIcon,
  hasAlts: false,
  isForMulti: false,
  execute: (dispatch, resources) => {
      dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
  },
};

export const msUserActionSet: MultiSelectMenuActionSet = [
    [
      msAdvancedAction, 
      msUserAttributes
    ]
];


export const msUserCommonActionFilter = new Set([ATTRIBUTES, API_DETAILS]);

