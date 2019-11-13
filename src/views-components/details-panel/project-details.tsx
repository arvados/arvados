// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
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
import * as CopyToClipboard from 'react-copy-to-clipboard';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { getTagValueLabel, getTagKeyLabel, Vocabulary } from '~/models/vocabulary';
import { getVocabulary } from "~/store/vocabulary/vocabulary-selectors";
import { Dispatch } from 'redux';

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
    vocabulary: Vocabulary;
}

interface ProjectDetailsComponentActionProps {
    onClick: () => void;
    onCopy: (message: string) => void;
}

const mapStateToProps = ({ properties }: RootState) => ({
    vocabulary: getVocabulary(properties),
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onClick: () => dispatch<any>(openProjectPropertiesDialog()),
    onCopy: (message: string) => dispatch(snackbarActions.OPEN_SNACKBAR({
        message,
        hideDuration: 2000,
        kind: SnackbarKind.SUCCESS
    }))
});

type ProjectDetailsComponentProps = ProjectDetailsComponentDataProps & ProjectDetailsComponentActionProps & WithStyles<CssRules>;

const ProjectDetailsComponent = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        ({ classes, project, onClick, vocabulary, onCopy }: ProjectDetailsComponentProps) => <div>
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
                    const label = `${getTagKeyLabel(k, vocabulary)}: ${getTagValueLabel(k, project.properties[k], vocabulary)}`;
                    return (
                        <CopyToClipboard key={k} text={label} onCopy={() => onCopy("Copied")}>
                            <Chip key={k} className={classes.tag} label={label} />
                        </CopyToClipboard>
                    );
                })
            }
        </div>
    ));
