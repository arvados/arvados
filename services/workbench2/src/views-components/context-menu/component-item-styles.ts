// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CustomStyleRulesCallback } from 'common/custom-theme';

export type ComponentCssRules = "toolbarIcon" | "toolbarButton" | "disabled";

export const componentItemStyles: CustomStyleRulesCallback<ComponentCssRules> = theme => ({
    toolbarIcon: {
        marginLeft: '1rem',
    },
    toolbarButton: {
        width: '3rem',
        height: '3rem',
    },
    disabled: {
        color: theme.palette.grey[400],
        cursor: 'none',
    },
});