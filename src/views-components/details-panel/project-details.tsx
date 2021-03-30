// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { openProjectPropertiesDialog } from '~/store/details-panel/details-panel-action';
import { ProjectIcon, RenameIcon, FilterGroupIcon } from '~/components/icon/icon';
import { ProjectResource } from '~/models/project';
import { formatDate } from '~/common/formatters';
import { ResourceKind } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "~/components/details-attribute/details-attribute";
import { RichTextEditorLink } from '~/components/rich-text-editor-link/rich-text-editor-link';
import { withStyles, StyleRulesCallback, WithStyles } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { Dispatch } from 'redux';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ResourceOwnerWithName } from '../data-explorer/renderers';
import { GroupClass } from "~/models/group";

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

type CssRules = 'tag' | 'editIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    },
    editIcon: {
        fontSize: '1.125rem',
        cursor: 'pointer'
    }
});

interface ProjectDetailsComponentDataProps {
    project: ProjectResource;
}

interface ProjectDetailsComponentActionProps {
    onClick: () => void;
}

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onClick: () => dispatch<any>(openProjectPropertiesDialog()),
});

type ProjectDetailsComponentProps = ProjectDetailsComponentDataProps & ProjectDetailsComponentActionProps & WithStyles<CssRules>;

const ProjectDetailsComponent = connect(null, mapDispatchToProps)(
    withStyles(styles)(
        ({ classes, project, onClick }: ProjectDetailsComponentProps) => <div>
            <DetailsAttribute label='Type' value={project.groupClass === GroupClass.FILTER ? 'Filter group' : resourceLabel(ResourceKind.PROJECT)} />
            <DetailsAttribute label='Owner' linkToUuid={project.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceOwnerWithName uuid={uuid} />} />
            <DetailsAttribute label='Last modified' value={formatDate(project.modifiedAt)} />
            <DetailsAttribute label='Created at' value={formatDate(project.createdAt)} />
            <DetailsAttribute label='Project UUID' linkToUuid={project.uuid} value={project.uuid} />
            <DetailsAttribute label='Description'>
                {project.description ?
                    <RichTextEditorLink
                        title={`Description of ${project.name}`}
                        content={project.description}
                        label='Show full description' />
                    : '---'
                }
            </DetailsAttribute>
            <DetailsAttribute label='Properties'>
                {project.groupClass !== GroupClass.FILTER ?
                    <div onClick={onClick}>
                        <RenameIcon className={classes.editIcon} />
                    </div>
                    : ''
                }
            </DetailsAttribute>
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
