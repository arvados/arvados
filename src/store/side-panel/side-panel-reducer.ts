// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sidePanelActions, SidePanelAction } from './side-panel-action';
import { SidePanelItem } from '~/components/side-panel/side-panel';
import { ProjectsIcon, ShareMeIcon, WorkflowIcon, RecentIcon, FavoriteIcon, TrashIcon } from "~/components/icon/icon";
import { Dispatch } from "redux";
import { push } from "react-router-redux";
import { favoritePanelActions } from "../favorite-panel/favorite-panel-action";
import { projectPanelActions } from "../project-panel/project-panel-action";
import { projectActions } from "../project/project-action";
import { getProjectUrl } from "~/models/project";
import { columns as projectPanelColumns } from "~/views/project-panel/project-panel";
import { columns as favoritePanelColumns } from "~/views/favorite-panel/favorite-panel";
import { columns as trashPanelColumns } from "~/views/trash-panel/trash-panel";
import { trashPanelActions } from "~/store/trash-panel/trash-panel-action";

export type SidePanelState = SidePanelItem[];

export const sidePanelReducer = (state: SidePanelState = sidePanelItems, action: SidePanelAction) => {
    return sidePanelActions.match(action, {
        TOGGLE_SIDE_PANEL_ITEM_OPEN: itemId =>
            state.map(it => ({...it, open: itemId === it.id && it.open === false})),
        default: () => state
    });
};

export enum SidePanelId {
    PROJECTS = "Projects",
    SHARED_WITH_ME = "SharedWithMe",
    WORKFLOWS = "Workflows",
    RECENT_OPEN = "RecentOpen",
    FAVORITES = "Favourites",
    TRASH = "Trash"
}

export const sidePanelItems = [
    {
        id: SidePanelId.PROJECTS,
        name: "Projects",
        url: "/projects",
        icon: ProjectsIcon,
        open: false,
        active: false,
        margin: true,
        openAble: true,
        activeAction: (dispatch: Dispatch, uuid: string) => {
            dispatch(push(getProjectUrl(uuid)));
            dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(uuid));
            dispatch(projectPanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
            dispatch(projectPanelActions.RESET_PAGINATION());
            dispatch(projectPanelActions.REQUEST_ITEMS());
        }
    },
    {
        id: SidePanelId.SHARED_WITH_ME,
        name: "Shared with me",
        url: "/shared",
        icon: ShareMeIcon,
        active: false,
        activeAction: (dispatch: Dispatch) => {
            dispatch(push("/shared"));
        }
    },
    {
        id: SidePanelId.WORKFLOWS,
        name: "Workflows",
        url: "/workflows",
        icon: WorkflowIcon,
        active: false,
        activeAction: (dispatch: Dispatch) => {
            dispatch(push("/workflows"));
        }
    },
    {
        id: SidePanelId.RECENT_OPEN,
        name: "Recent open",
        url: "/recent",
        icon: RecentIcon,
        active: false,
        activeAction: (dispatch: Dispatch) => {
            dispatch(push("/recent"));
        }
    },
    {
        id: SidePanelId.FAVORITES,
        name: "Favorites",
        url: "/favorites",
        icon: FavoriteIcon,
        active: false,
        activeAction: (dispatch: Dispatch) => {
            dispatch(push("/favorites"));
            dispatch(favoritePanelActions.SET_COLUMNS({ columns: favoritePanelColumns }));
            dispatch(favoritePanelActions.RESET_PAGINATION());
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        }
    },
    {
        id: SidePanelId.TRASH,
        name: "Trash",
        url: "/trash",
        icon: TrashIcon,
        active: false,
        activeAction: (dispatch: Dispatch) => {
            dispatch(push("/trash"));
            dispatch(trashPanelActions.SET_COLUMNS({ columns: trashPanelColumns }));
            dispatch(trashPanelActions.RESET_PAGINATION());
            dispatch(trashPanelActions.REQUEST_ITEMS());
        }
    }
];
