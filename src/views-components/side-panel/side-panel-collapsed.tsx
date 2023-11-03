// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement } from 'react'
import { connect } from 'react-redux'
import { ProjectsIcon, ProcessIcon, FavoriteIcon, ShareMeIcon, TrashIcon, PublicFavoriteIcon, GroupsIcon } from 'components/icon/icon'
import { List, ListItem, Tooltip } from '@material-ui/core'
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles'
import { ArvadosTheme } from 'common/custom-theme'
import { navigateTo } from 'store/navigation/navigation-action'
import { RootState } from 'store/store'
import { Dispatch } from 'redux'
import {
    navigateToSharedWithMe,
    navigateToPublicFavorites,
    navigateToFavorites,
    navigateToGroups,
    navigateToAllProcesses,
    navigateToTrash,
} from 'store/navigation/navigation-action'
import { RouterAction } from 'react-router-redux'

type CssRules = 'root' | 'icon'

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {},
    icon: {
        color: theme.customs.colors.grey700,
    },
})

enum SidePanelCollapsedCategory {
    PROJECTS = 'Home Projects',
    SHARED_WITH_ME = 'Shared with me',
    PUBLIC_FAVORITES = 'Public Favorites',
    FAVORITES = 'My Favorites',
    GROUPS = 'Groups',
    ALL_PROCESSES = 'All Processes',
    TRASH = 'Trash',
}

type TCollapsedCategory = {
    name: SidePanelCollapsedCategory
    icon: ReactElement
    navTarget: RouterAction | ''
}

const sidePanelCollapsedCategories: TCollapsedCategory[] = [
    {
        name: SidePanelCollapsedCategory.PROJECTS,
        icon: <ProjectsIcon />,
        navTarget: '',
    },
    {
        name: SidePanelCollapsedCategory.SHARED_WITH_ME,
        icon: <ShareMeIcon />,
        navTarget: navigateToSharedWithMe,
    },
    {
        name: SidePanelCollapsedCategory.PUBLIC_FAVORITES,
        icon: <PublicFavoriteIcon />,
        navTarget: navigateToPublicFavorites,
    },
    {
        name: SidePanelCollapsedCategory.FAVORITES,
        icon: <FavoriteIcon />,
        navTarget: navigateToFavorites,
    },
    {
        name: SidePanelCollapsedCategory.GROUPS,
        icon: <GroupsIcon />,
        navTarget: navigateToGroups,
    },
    {
        name: SidePanelCollapsedCategory.ALL_PROCESSES,
        icon: <ProcessIcon />,
        navTarget: navigateToAllProcesses,
    },
    {
        name: SidePanelCollapsedCategory.TRASH,
        icon: <TrashIcon />,
        navTarget: navigateToTrash,
    },
]

const mapStateToProps = (state: RootState) => {
    return {
        user: state.auth.user,
    }
}

const mapDispatchToProps = (dispatch: Dispatch) => {
    return {
        navToHome: (navTarget) => dispatch<any>(navigateTo(navTarget)),
        navTo: (navTarget) => dispatch<any>(navTarget),
    }
}

export const SidePanelCollapsed = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(({ classes, user, navToHome, navTo }: WithStyles & any) => {

        const handleClick = (cat: TCollapsedCategory) => {
            if (cat.name === SidePanelCollapsedCategory.PROJECTS) navToHome(user.uuid)
            else navTo(cat.navTarget)
        }

        return (
            <List>
                {sidePanelCollapsedCategories.map(cat => (
                    <ListItem
                        key={cat.name}
                        className={classes.icon}
                        onClick={() => handleClick(cat)}
                    >
                        <Tooltip
                            title={cat.name}
                            disableFocusListener
                        >
                            {cat.icon}
                        </Tooltip>
                    </ListItem>
                ))}
            </List>
        )
    })
)
