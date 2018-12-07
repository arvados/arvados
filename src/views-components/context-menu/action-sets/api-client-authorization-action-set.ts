// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    openApiClientAuthorizationAttributesDialog,
    openApiClientAuthorizationRemoveDialog
} from '~/store/api-client-authorizations/api-client-authorizations-actions';
import { openAdvancedTabDialog } from '~/store/advanced-tab/advanced-tab';
import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, AttributesIcon } from "~/components/icon/icon";

export const apiClientAuthorizationActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openApiClientAuthorizationAttributesDialog(uuid));
    }
}, {
    name: "Advanced",
    icon: AdvancedIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openAdvancedTabDialog(uuid));
    }
}, {
    name: "Remove",
    icon: RemoveIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openApiClientAuthorizationRemoveDialog(uuid));
    }
}]];
