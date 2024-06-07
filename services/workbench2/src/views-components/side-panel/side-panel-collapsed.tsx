// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement } from 'react'
import { connect } from 'react-redux'
import { ProjectsIcon, ProcessIcon, FavoriteIcon, ShareMeIcon, TrashIcon, PublicFavoriteIcon, GroupsIcon, ResourceIcon } from 'components/icon/icon'
import { TerminalIcon } from 'components/icon/icon'
import { IconButton, List, ListItem, Tooltip } from '@material-ui/core'
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles, withStyles } from '@material-ui/core/styles'
import { ArvadosTheme } from 'common/custom-theme'
import { navigateTo, navigateToInstanceTypes } from 'store/navigation/navigation-action'
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
import { navigateToUserVirtualMachines } from 'store/navigation/navigation-action'
import { RouterAction } from 'react-router-redux'
import { User } from 'models/user'

type CssRules = 'button' | 'unselected' | 'selected'

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    button: {
        width: '40px',
        height: '40px',
        paddingLeft: '-2rem',
        marginLeft: '-0.6rem',
        marginBottom: '-1rem'
    },
    unselected: {
        color: theme.customs.colors.grey600,
    },
    selected: {
        color: theme.palette.primary.main,
    },
})

enum SidePanelCollapsedCategory {
    PROJECTS = 'Home Projects',
    FAVORITES = 'My Favorites',
    PUBLIC_FAVORITES = 'Public Favorites',
    SHARED_WITH_ME = 'Shared with me',
    ALL_PROCESSES = 'All Processes',
    INSTANCE_TYPES = 'Instance Types',
    SHELL_ACCESS = 'Shell Access',
    GROUPS = 'Groups',
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
        name: SidePanelCollapsedCategory.FAVORITES,
        icon: <FavoriteIcon />,
        navTarget: navigateToFavorites,
    },
    {
        name: SidePanelCollapsedCategory.PUBLIC_FAVORITES,
        icon: <PublicFavoriteIcon />,
        navTarget: navigateToPublicFavorites,
    },
    {
        name: SidePanelCollapsedCategory.SHARED_WITH_ME,
        icon: <ShareMeIcon />,
        navTarget: navigateToSharedWithMe,
    },
    {
        name: SidePanelCollapsedCategory.ALL_PROCESSES,
        icon: <ProcessIcon />,
        navTarget: navigateToAllProcesses,
    },
    {
        name: SidePanelCollapsedCategory.INSTANCE_TYPES,
        icon: <ResourceIcon />,
        navTarget: navigateToInstanceTypes,
    },
    {
        name: SidePanelCollapsedCategory.SHELL_ACCESS,
        icon: <TerminalIcon />,
        navTarget: navigateToUserVirtualMachines,
    },
    {
        name: SidePanelCollapsedCategory.GROUPS,
        icon: <GroupsIcon style={{marginLeft: '2px', scale: '85%'}}/>,
        navTarget: navigateToGroups,
    },
    {
        name: SidePanelCollapsedCategory.TRASH,
        icon: <TrashIcon />,
        navTarget: navigateToTrash,
    },
]

type SidePanelCollapsedProps = {
    user: User;
    selectedPath: string;
    navToHome: (uuid: string) => void;
    navTo: (navTarget: RouterAction | '') => void;
};

const mapStateToProps = ({auth, properties }: RootState) => {
        return {
            user: auth.user,
            selectedPath: properties.breadcrumbs
                ? properties.breadcrumbs[0].label
                : SidePanelCollapsedCategory.PROJECTS,
        }
}

const mapDispatchToProps = (dispatch: Dispatch) => {
    return {
        navToHome: (navTarget) => dispatch<any>(navigateTo(navTarget)),
        navTo: (navTarget) => dispatch<any>(navTarget),
    }
}

export const SidePanelCollapsed = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(({ classes, user, selectedPath, navToHome, navTo }: WithStyles & SidePanelCollapsedProps) => {

        const handleClick = (cat: TCollapsedCategory) => {
            if (cat.name === SidePanelCollapsedCategory.PROJECTS) navToHome(user.uuid)
            else navTo(cat.navTarget)
        }

        const { button, unselected, selected } = classes
        return (
            <List data-cy='side-panel-collapsed'>
                {sidePanelCollapsedCategories.map((cat) => (
                    <ListItem
                        key={cat.name}
                        data-cy={`collapsed-${cat.name.toLowerCase().replace(/\s+/g, '-')}`}
                        onClick={() => handleClick(cat)}
                        >
                        <Tooltip
                            className={selectedPath === cat.name ? selected : unselected}
                            title={cat.name}
                            disableFocusListener
                            >
                            <IconButton className={button}>{cat.icon}</IconButton>
                        </Tooltip>
                    </ListItem>
                ))}
            </List>
        );
    })
)
