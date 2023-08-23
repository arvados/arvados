// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { ProjectIcon, RenameIcon, FilterGroupIcon } from 'components/icon/icon';
import { ProjectResource } from 'models/project';
import { formatDate } from 'common/formatters';
import { ResourceKind } from 'models/resource';
import { resourceLabel } from 'common/labels';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { RichTextEditorLink } from 'components/rich-text-editor-link/rich-text-editor-link';
import { withStyles, StyleRulesCallback, WithStyles, Button } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { Dispatch } from 'redux';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ResourceWithName } from '../data-explorer/renderers';
import { GroupClass } from "models/group";
import { openProjectUpdateDialog, ProjectUpdateFormDialogData } from 'store/projects/project-update-actions';
import { RootState } from 'store/store';
import { ResourcesState } from 'store/resources/resources';
import { resourceIsFrozen } from 'common/frozen-resources';

export class ProjectDetails extends DetailsData<ProjectResource> {
    getIcon(className?: string) {
        if (this.item.groupClass === GroupClass.FILTER) {
            return <FilterGroupIcon className={className} />;
        }
        return <ProjectIcon className={className} />;
    }

    getDetails() {
        return <ProjectDetailsComponent project={this.item} />;
    }
}

type CssRules = 'tag' | 'editIcon' | 'editButton';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit / 2,
        marginBottom: theme.spacing.unit / 2,
    },
    editIcon: {
        paddingRight: theme.spacing.unit / 2,
        fontSize: '1.125rem',
    },
    editButton: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
});

interface ProjectDetailsComponentDataProps {
    project: ProjectResource;
}

interface ProjectDetailsComponentActionProps {
    onClick: (prj: ProjectUpdateFormDialogData) => () => void;
}

const mapStateToProps = (state: RootState): { resources: ResourcesState } => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onClick: (prj: ProjectUpdateFormDialogData) =>
        () => dispatch<any>(openProjectUpdateDialog(prj)),
});

type ProjectDetailsComponentProps = ProjectDetailsComponentDataProps & ProjectDetailsComponentActionProps & WithStyles<CssRules>;

const ProjectDetailsComponent = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        ({ classes, project, resources, onClick }: ProjectDetailsComponentProps & { resources: ResourcesState }) => <div>
            {project.groupClass !== GroupClass.FILTER ?
                <Button onClick={onClick({
                    uuid: project.uuid,
                    name: project.name,
                    description: project.description,
                    properties: project.properties,
                })}
                    disabled={resourceIsFrozen(project, resources)}
                    className={classes.editButton} variant='contained'
                    data-cy='details-panel-edit-btn' color='primary' size='small'>
                    <RenameIcon className={classes.editIcon} /> Edit
                </Button>
                : ''
            }
            <DetailsAttribute label='Type' value={project.groupClass === GroupClass.FILTER ? 'Filter group' : resourceLabel(ResourceKind.PROJECT)} />
            <DetailsAttribute label='Owner' linkToUuid={project.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
            <DetailsAttribute label='Last modified' value={formatDate(project.modifiedAt)} />
            <DetailsAttribute label='Created at' value={formatDate(project.createdAt)} />
            <DetailsAttribute label='UUID' linkToUuid={project.uuid} value={project.uuid} />
            <DetailsAttribute label='Description'>
                {project.description ?
                    <RichTextEditorLink
                        title={`Description of ${project.name}`}
                        content={project.description}
                        label='Show full description' />
                    : '---'
                }
            </DetailsAttribute>
            <DetailsAttribute label='Properties' />
            {
                Object.keys(project.properties).map(k =>
                    Array.isArray(project.properties[k])
                        ? project.properties[k].map((v: string) =>
                            getPropertyChip(k, v, undefined, classes.tag))
                        : getPropertyChip(k, project.properties[k], undefined, classes.tag)
                )
            }
        </div>
    ));
