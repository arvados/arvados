// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

import actions, { SidePanelAction } from './side-panel-action';
import { SidePanelItem } from '../../components/side-panel/side-panel';

export type SidePanelState = SidePanelItem[];

const sidePanelReducer = (state: SidePanelState = sidePanelData, action: SidePanelAction) => {
    return actions.match(action, {
        TOGGLE_SIDE_PANEL_ITEM_OPEN: () => {
            const sidePanel = _.cloneDeep(state);
            sidePanel[0].open = !sidePanel[0].open;
            return sidePanel;
        },
        TOGGLE_SIDE_PANEL_ITEM_ACTIVE: itemId => {
            const sidePanel = _.cloneDeep(state);
            resetSidePanelActivity(sidePanel);
            sidePanel.map(it => {
                if (it.id === itemId) {
                    it.active = true;
                }
            });
            resetProjectsCollapse(sidePanel); 
            return sidePanel;
        },
        default: () => state
    });
};

export const sidePanelData = [
    {
        id: "1",
        name: "Projects",
        icon: "fas fa-th fa-fw",
        open: false,
        active: false,
    },
    {
        id: "2",
        name: "Shared with me",
        icon: "fas fa-users fa-fw",
        active: false,
    },
    {
        id: "3",
        name: "Workflows",
        icon: "fas fa-cogs fa-fw",
        active: false,
    },
    {
        id: "4",
        name: "Recent open",
        icon: "icon-time fa-fw",
        active: false,
    },
    {
        id: "5",
        name: "Favorites",
        icon: "fas fa-star fa-fw",
        active: false,
    },
    {
        id: "6",
        name: "Trash",
        icon: "fas fa-trash-alt fa-fw",
        active: false,
    }
];

function resetSidePanelActivity(sidePanel: SidePanelItem[]) {
    for (const t of sidePanel) {
        t.active = false;
    }
}

function resetProjectsCollapse(sidePanel: SidePanelItem[]) {
    if (!sidePanel[0].active) {
        sidePanel[0].open = false;
    }
}

export default sidePanelReducer;
