// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from '~/store/store';
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { ProjectResource } from '~/models/project';
import { PROJECT_PROPERTIES_DIALOG_NAME, deleteProjectProperty } from '~/store/details-panel/details-panel-action';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Chip, withStyles, StyleRulesCallback, WithStyles } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { ProjectPropertiesForm } from '~/views-components/project-properties-dialog/project-properties-form';
import { getResource } from '~/store/resources/resources';
import * as CopyToClipboard from 'react-copy-to-clipboard';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { getTagValueLabel, getTagKeyLabel, Vocabulary } from '~/models/vocabulary';
import { getVocabulary } from "~/store/vocabulary/vocabulary-selectors";

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface ProjectPropertiesDialogDataProps {
    project: ProjectResource;
    vocabulary: Vocabulary;
}

interface ProjectPropertiesDialogActionProps {
    handleDelete: (key: string) => void;
    onCopy: (message: string) => void;
}

const mapStateToProps = ({ detailsPanel, resources, properties }: RootState): ProjectPropertiesDialogDataProps => ({
    project: getResource(detailsPanel.resourceUuid)(resources) as ProjectResource,
    vocabulary: getVocabulary(properties),
});

const mapDispatchToProps = (dispatch: Dispatch): ProjectPropertiesDialogActionProps => ({
    handleDelete: (key: string) => dispatch<any>(deleteProjectProperty(key)),
    onCopy: (message: string) => dispatch(snackbarActions.OPEN_SNACKBAR({
                message,
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }))
});

type ProjectPropertiesDialogProps =  ProjectPropertiesDialogDataProps & ProjectPropertiesDialogActionProps & WithDialogProps<{}> & WithStyles<CssRules>;

export const ProjectPropertiesDialog = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
    withDialog(PROJECT_PROPERTIES_DIALOG_NAME)(
        ({ classes, open, closeDialog, handleDelete, onCopy, project, vocabulary }: ProjectPropertiesDialogProps) =>
            <Dialog open={open}
                onClose={closeDialog}
                fullWidth
                maxWidth='sm'>
                <DialogTitle>Properties</DialogTitle>
                <DialogContent>
                    <ProjectPropertiesForm />
                    {project && project.properties &&
                        Object.keys(project.properties).map(k => {
                            const label = `${getTagKeyLabel(k, vocabulary)}: ${getTagValueLabel(k, project.properties[k], vocabulary)}`;
                            return (
                                <CopyToClipboard key={k} text={label} onCopy={() => onCopy("Copied")}>
                                    <Chip key={k} className={classes.tag}
                                        onDelete={() => handleDelete(k)}
                                        label={label} />
                                </CopyToClipboard>
                            );
                        })
                    }
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={closeDialog}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
)));