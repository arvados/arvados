// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sidePanelReducer } from "./side-panel-reducer";
import { sidePanelActions } from "./side-panel-action";
import { ProjectsIcon } from "~/components/icon/icon";

describe('side-panel-reducer', () => {
    it('should open side-panel item', () => {
        const initialState = [
            {
                id: "1",
                name: "Projects",
                url: "/projects",
                icon: ProjectsIcon,
                open: false
            }
        ];
        const project = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: true
            }
        ];

        const state = sidePanelReducer(initialState, sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(initialState[0].id));
        expect(state).toEqual(project);
    });
});
