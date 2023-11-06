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

type CssRules = 'root' | 'unselected' | 'selected'

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {},
    unselected: {
        color: theme.customs.colors.grey700,
    },
    selected: {
        color: theme.palette.primary.main,
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

const mapStateToProps = ({auth, properties }: RootState) => {
    return {
        user: auth.user,
        selectedPath: properties.breadcrumbs ? properties.breadcrumbs[0].label : SidePanelCollapsedCategory.PROJECTS,
    }
}

const mapDispatchToProps = (dispatch: Dispatch) => {
    return {
        navToHome: (navTarget) => dispatch<any>(navigateTo(navTarget)),
        navTo: (navTarget) => dispatch<any>(navTarget),
    }
}

export const SidePanelCollapsed = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(({ classes, user, selectedPath, navToHome, navTo }: WithStyles & any) => {

        const handleClick = (cat: TCollapsedCategory) => {
            if (cat.name === SidePanelCollapsedCategory.PROJECTS) navToHome(user.uuid)
            else navTo(cat.navTarget)
        }

        const { root, unselected, selected } = classes
        return (
            <List data-cy="side-panel-collapsed" className={root}>
                {sidePanelCollapsedCategories.map((cat) => (
                    <ListItem
                        key={cat.name}
                        data-cy={`collapsed-${cat.name.toLowerCase().replace(/\s+/g, '-')}`}
                        className={selectedPath === cat.name ? selected : unselected}
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
