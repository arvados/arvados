// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { connect } from "react-redux";
import { ProjectTree } from '~/views-components/project-tree/project-tree';
import { SidePanel, SidePanelItem } from '~/components/side-panel/side-panel';
import { ArvadosTheme } from '~/common/custom-theme';
import { RootState } from '~/store/store';
import { TreeItem } from '~/components/tree/tree';
import { ProjectResource } from '~/models/project';
import { sidePanelActions } from '../../store/side-panel/side-panel-action';
import { Dispatch } from 'redux';
import { projectActions } from '~/store/project/project-action';
import { navigateToResource } from '../../store/navigation/navigation-action';
import { openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';


const DRAWER_WITDH = 240;

type CssRules = 'drawerPaper' | 'toolbar';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    drawerPaper: {
        position: 'relative',
        width: DRAWER_WITDH,
        display: 'flex',
        flexDirection: 'column',
    },
    toolbar: theme.mixins.toolbar
});

interface NavigationPanelDataProps {
    projects: Array<TreeItem<ProjectResource>>;
    sidePanelItems: SidePanelItem[];
}

interface NavigationPanelActionProps {
    toggleSidePanelOpen: (panelItemId: string) => void;
    toggleSidePanelActive: (panelItemId: string) => void;
    toggleProjectOpen: (projectUuid: string) => void;
    toggleProjectActive: (projectUuid: string) => void;
    openRootContextMenu: (event: React.MouseEvent<any>) => void;
    openProjectContextMenu: (event: React.MouseEvent<any>, item: TreeItem<ProjectResource>) => void;
}

type NavigationPanelProps = NavigationPanelDataProps & NavigationPanelActionProps & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): NavigationPanelDataProps => ({
    projects: state.projects.items,
    sidePanelItems: state.sidePanel
});

const mapDispatchToProps = (dispatch: Dispatch): NavigationPanelActionProps => ({
    toggleSidePanelOpen: panelItemId => {
        dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(panelItemId));
    },
    toggleSidePanelActive: panelItemId => {
        dispatch(projectActions.RESET_PROJECT_TREE_ACTIVITY(panelItemId));

        // const panelItem = this.props.sidePanelItems.find(it => it.id === itemId);
        // if (panelItem && panelItem.activeAction) {
        //     panelItem.activeAction(this.props.dispatch, this.props.authService.getUuid());
        // }
    },
    toggleProjectOpen: projectUuid => {
        dispatch<any>(navigateToResource(projectUuid));
    },
    toggleProjectActive: projectUuid => {
        dispatch<any>(navigateToResource(projectUuid));
    },
    openRootContextMenu: event => {
        dispatch<any>(openContextMenu(event, {
            uuid: "",
            name: "",
            kind: ContextMenuKind.ROOT_PROJECT
        }));
    },
    openProjectContextMenu: (event, item) => {
        dispatch<any>(openContextMenu(event, {
            uuid: item.data.uuid,
            name: item.data.name,
            kind: ContextMenuKind.PROJECT
        }));
    }
});

export const NavigationPanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        ({ classes, sidePanelItems, projects, ...actions }: NavigationPanelProps) => <Drawer
            variant="permanent"
            classes={{ paper: classes.drawerPaper }}>
            <div className={classes.toolbar} />
            <SidePanel
                toggleOpen={actions.toggleSidePanelOpen}
                toggleActive={actions.toggleSidePanelOpen}
                sidePanelItems={sidePanelItems}
                onContextMenu={actions.openRootContextMenu}>
                <ProjectTree
                    projects={projects}
                    toggleOpen={actions.toggleProjectOpen}
                    onContextMenu={actions.openProjectContextMenu}
                    toggleActive={actions.toggleProjectActive} />
            </SidePanel>
        </Drawer>
    )
);
