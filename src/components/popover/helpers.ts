// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PopoverOrigin } from "@material-ui/core/Popover";

export const mockAnchorFromMouseEvent = (event: React.MouseEvent<HTMLElement>) => {
    const el = document.createElement('div');
    const clientRect = {
        left: event.clientX,
        right: event.clientX,
        top: event.clientY,
        bottom: event.clientY,
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