// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PopoverOrigin } from "@material-ui/core/Popover";

export const createAnchorAt = (position: {x: number, y: number}) => {
    const el = document.createElement('div');
    const clientRect = {
        left: position.x,
        right: position.x,
        top: position.y,
        bottom: position.y,
        width: 0,
        height: 0
    };
    el.getBoundingClientRect = () => clientRect;
    return el;
};

export const DefaultTransformOrigin: PopoverOrigin = {
    vertical: "top",
    horizontal: "right",
};