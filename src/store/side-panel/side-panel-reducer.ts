// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { sidePanelActions, SidePanelAction } from './side-panel-action';
import { SidePanelItem } from '~/components/side-panel/side-panel';
import { ProjectsIcon, ShareMeIcon, WorkflowIcon, RecentIcon, FavoriteIcon, TrashIcon } from "~/components/icon/icon";
import { Dispatch } from "redux";
import { push } from "react-router-redux";
import { favoritePanelActions } from "../favorite-panel/favorite-panel-action";
import { projectPanelActions } from "../project-panel/project-panel-action";
import { projectActions } from "../project/project-action";
import { getProjectUrl } from "../../models/project";

export type SidePanelState = SidePanelItem[];

export const sidePanelReducer = (state: SidePanelState = sidePanelData, action: SidePanelAction) => {
    if (state.length === 0) {
        return sidePanelData;
    } else {
        return sidePanelActions.match(action, {
            TOGGLE_SIDE_PANEL_ITEM_OPEN: itemId =>
                state.map(it => ({...it, open: itemId === it.id && it.open === false})),
            TOGGLE_SIDE_PANEL_ITEM_ACTIVE: itemId => {
                const sidePanel = _.cloneDeep(state);
                resetSidePanelActivity(sidePanel);
                sidePanel.forEach(it => {
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
    PROJECTS = "Projects",
    SHARED_WITH_ME = "SharedWithMe",
    WORKFLOWS = "Workflows",
    RECENT_OPEN = "RecentOpen",
    FAVORITES = "Favourites",
    TRASH = "Trash"
}

export const sidePanelData = [
    {
        id: SidePanelIdentifiers.PROJECTS,
        name: "Projects",
        icon: ProjectsIcon,
        open: false,
        active: false,
        margin: true,
        openAble: true,
        activeAction: (dispatch: Dispatch, uuid: string) => {
            dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(uuid));
            dispatch(push(getProjectUrl(uuid)));
            dispatch(projectPanelActions.RESET_PAGINATION());
            dispatch(projectPanelActions.REQUEST_ITEMS()); 
        }
    },
    {
        id: SidePanelIdentifiers.SHARED_WITH_ME,
        name: "Shared with me",
        icon: ShareMeIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.WORKFLOWS,
        name: "Workflows",
        icon: WorkflowIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.RECENT_OPEN,
        name: "Recent open",
        icon: RecentIcon,
        active: false,
    },
    {
        id: SidePanelIdentifiers.FAVORITES,
        name: "Favorites",
        icon: FavoriteIcon,
        active: false,
        activeAction: (dispatch: Dispatch) => {
            dispatch(push("/favorites"));
            dispatch(favoritePanelActions.RESET_PAGINATION());
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        }
    },
    {
        id: SidePanelIdentifiers.TRASH,
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
