// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { sidePanelActions, SidePanelAction } from './side-panel-action';
import { SidePanelItem } from '../../components/side-panel/side-panel';
import { ProjectsIcon, ShareMeIcon, WorkflowIcon, RecentIcon, FavoriteIcon, TrashIcon } from "../../components/icon/icon";

export type SidePanelState = SidePanelItem[];

export const sidePanelReducer = (state: SidePanelState = sidePanelData, action: SidePanelAction) => {
    if (state.length === 0) {
        return sidePanelData;
    } else {
        return sidePanelActions.match(action, {
            TOGGLE_SIDE_PANEL_ITEM_OPEN: itemId => state.map(it => itemId === it.id && it.open === false ? {...it, open: true} : {...it, open: false}),
            TOGGLE_SIDE_PANEL_ITEM_ACTIVE: itemId => {
                const sidePanel = _.cloneDeep(state);
                resetSidePanelActivity(sidePanel);
                sidePanel.map(it => {
                    if (it.id === itemId) {
                        it.active = true;
                    }
                });
                return sidePanel;
            },
            RESET_SIDE_PANEL_ACTIVITY: () => {
                const sidePanel = _.cloneDeep(state);
                resetSidePanelActivity(sidePanel);
                return sidePanel;
            },
            default: () => state
        });
    }
};

export enum SidePanelIdentifiers {
    Projects = "Projects",
    SharedWithMe = "SharedWithMe",
    Workflows = "Workflows",
    RecentOpen = "RecentOpen",
    Favourites = "Favourites",
    Trash = "Trash"
}

export const sidePanelData = [
    {
        id: SidePanelIdentifiers.Projects,
        name: "Projects",
        icon: ProjectsIcon,
        open: false,
        active: false,
        margin: true,
        openAble: true
    },
    {
        id: SidePanelIdentifiers.SharedWithMe,
        name: "Shared with me",
        icon: ShareMeIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.Workflows,
        name: "Workflows",
        icon: WorkflowIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.RecentOpen,
        name: "Recent open",
        icon: RecentIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.Favourites,
        name: "Favorites",
        icon: FavoriteIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.Trash,
        name: "Trash",
        icon: TrashIcon,
        active: false,
    }
];

function resetSidePanelActivity(sidePanel: SidePanelItem[]) {
    for (const t of sidePanel) {
        t.active = false;
    }
}
