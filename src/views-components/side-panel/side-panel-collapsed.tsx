// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement } from 'react'
import { ProjectIcon, ProcessIcon, FavoriteIcon, ShareMeIcon, TrashIcon, PublicFavoriteIcon, GroupsIcon } from 'components/icon/icon'
import { List, ListItem, Tooltip } from '@material-ui/core'
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles'
import { ArvadosTheme } from 'common/custom-theme'
import { navigateTo } from 'store/navigation/navigation-action'

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
    TRASH = 'Trash',
    ALL_PROCESSES = 'All Processes',
    GROUPS = 'Groups',
}

type TCollapsedCategory = {
    name: SidePanelCollapsedCategory
    icon: ReactElement
    navTarget: string
}

const sidePanelCollapsedCategories: TCollapsedCategory[] = [
    {
        name: SidePanelCollapsedCategory.PROJECTS,
        icon: <ProjectIcon />,
        navTarget: 'foo',
    },
    {
        name: SidePanelCollapsedCategory.SHARED_WITH_ME,
        icon: <ShareMeIcon />,
        navTarget: 'foo',
    },
    {
        name: SidePanelCollapsedCategory.PUBLIC_FAVORITES,
        icon: <PublicFavoriteIcon />,
        navTarget: 'public-favorites',
    },
    {
        name: SidePanelCollapsedCategory.FAVORITES,
        icon: <FavoriteIcon />,
        navTarget: 'foo',
    },
    {
        name: SidePanelCollapsedCategory.GROUPS,
        icon: <GroupsIcon />,
        navTarget: 'foo',
    },
    {
        name: SidePanelCollapsedCategory.ALL_PROCESSES,
        icon: <ProcessIcon />,
        navTarget: 'foo',
    },
    {
        name: SidePanelCollapsedCategory.TRASH,
        icon: <TrashIcon />,
        navTarget: 'foo',
    },
]

export const SidePanelCollapsed = withStyles(styles)(({ classes }: WithStyles) => {

    const handleClick = (navTarget: string) => {
        console.log(navTarget)
        navigateTo(navTarget)
    }

    return (
        <List>
            {sidePanelCollapsedCategories.map(cat => (
                <ListItem
                    key={cat.name}
                    className={classes.icon}
                    onClick={()=> handleClick(cat.navTarget)}
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
