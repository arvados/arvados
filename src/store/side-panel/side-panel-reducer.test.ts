// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sidePanelReducer } from "./side-panel-reducer";
import { sidePanelActions } from "./side-panel-action";
import { ProjectsIcon } from "../../components/icon/icon";

describe('side-panel-reducer', () => {

    it('should toggle activity on side-panel', () => {
        const initialState = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: false,
                active: false,
            }
        ];
        const project = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: false,
                active: true,
            }
        ];

        const state = sidePanelReducer(initialState, sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(initialState[0].id));
        expect(state).toEqual(project);
    });

    it('should open side-panel item', () => {
        const initialState = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: false,
                active: false,
            }
        ];
        const project = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: true,
                active: false,
            }
        ];

        const state = sidePanelReducer(initialState, sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(initialState[0].id));
        expect(state).toEqual(project);
    });

    it('should remove activity on side-panel item', () => {
        const initialState = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: false,
                active: true,
            }
        ];
        const project = [
            {
                id: "1",
                name: "Projects",
                icon: ProjectsIcon,
                open: false,
                active: false,
            }
        ];

        const state = sidePanelReducer(initialState, sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(initialState[0].id));
        expect(state).toEqual(project);
    });
});
