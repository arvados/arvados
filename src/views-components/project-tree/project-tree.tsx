// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ReactElement } from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Tree, TreeItem } from '~/components/tree/tree';
import { ProjectResource } from '~/models/project';
import { ProjectIcon } from '~/components/icon/icon';
import { ArvadosTheme } from '~/common/custom-theme';
import { ListItemTextIcon } from '~/components/list-item-text-icon/list-item-text-icon';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        marginLeft: `${theme.spacing.unit * 1.5}px`,
    }
});

export interface ProjectTreeProps<T> {
    projects: Array<TreeItem<ProjectResource>>;
    toggleOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    toggleActive: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: TreeItem<ProjectResource>) => void;
}

export const ProjectTree = withStyles(styles)(
    class ProjectTreeGeneric<T> extends React.Component<ProjectTreeProps<T> & WithStyles<CssRules>> {
        render(): ReactElement<any> {
            const { classes, projects, toggleOpen, toggleActive, onContextMenu } = this.props;
            return (
                <div className={classes.root}>
                    <Tree items={projects}
                        onContextMenu={onContextMenu}
                        toggleItemOpen={toggleOpen}
                        toggleItemActive={toggleActive}
                        render={
                            (project: TreeItem<ProjectResource>) =>
                                <ListItemTextIcon
                                    icon={ProjectIcon}
                                    name={project.data.name}
                                    isActive={project.active}
                                    hasMargin={true} />
                        } />
                </div>
            );
        }
    }
);
