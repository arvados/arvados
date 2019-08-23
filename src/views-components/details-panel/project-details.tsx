// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { openProjectPropertiesDialog } from '~/store/details-panel/details-panel-action';
import { ProjectIcon, RenameIcon } from '~/components/icon/icon';
import { ProjectResource } from '~/models/project';
import { formatDate } from '~/common/formatters';
import { ResourceKind } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "~/components/details-attribute/details-attribute";
import { RichTextEditorLink } from '~/components/rich-text-editor-link/rich-text-editor-link';
import { withStyles, StyleRulesCallback, Chip, WithStyles } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';

export class ProjectDetails extends DetailsData<ProjectResource> {
    getIcon(className?: string) {
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

const mapDispatchToProps = ({ onClick: openProjectPropertiesDialog });

type ProjectDetailsComponentProps = ProjectDetailsComponentDataProps & ProjectDetailsComponentActionProps & WithStyles<CssRules>;

const ProjectDetailsComponent = connect(null, mapDispatchToProps)(
    withStyles(styles)(
        ({ classes, project, onClick }: ProjectDetailsComponentProps) => <div>
            <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROJECT)} />
            {/* Missing attr */}
            <DetailsAttribute label='Size' value='---' />
            <DetailsAttribute label='Owner' linkToUuid={project.ownerUuid} lowercaseValue={true} />
            <DetailsAttribute label='Last modified' value={formatDate(project.modifiedAt)} />
            <DetailsAttribute label='Created at' value={formatDate(project.createdAt)} />
            <DetailsAttribute label='Project UUID' linkToUuid={project.uuid} value={project.uuid} />
            {/* Missing attr */}
            {/*<DetailsAttribute label='File size' value='1.4 GB' />*/}
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
                <div onClick={onClick}>
                    <RenameIcon className={classes.editIcon} />
                </div>
            </DetailsAttribute>
            {
                Object.keys(project.properties).map(k => {
                    return <Chip key={k} className={classes.tag} label={`${k}: ${project.properties[k]}`} />;
                })
            }
        </div>
    ));
