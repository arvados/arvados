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
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, withStyles, StyleRulesCallback, WithStyles } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { ProjectPropertiesForm } from '~/views-components/project-properties-dialog/project-properties-form';
import { getResource } from '~/store/resources/resources';
import { getPropertyChip } from "../resource-properties-form/property-chip";

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface ProjectPropertiesDialogDataProps {
    project: ProjectResource;
}

interface ProjectPropertiesDialogActionProps {
    handleDelete: (key: string, value: string) => void;
}

const mapStateToProps = ({ detailsPanel, resources, properties }: RootState): ProjectPropertiesDialogDataProps => ({
    project: getResource(detailsPanel.resourceUuid)(resources) as ProjectResource,
});

const mapDispatchToProps = (dispatch: Dispatch): ProjectPropertiesDialogActionProps => ({
    handleDelete: (key: string, value: string) => () => dispatch<any>(deleteProjectProperty(key, value)),
});

type ProjectPropertiesDialogProps = ProjectPropertiesDialogDataProps & ProjectPropertiesDialogActionProps & WithDialogProps<{}> & WithStyles<CssRules>;

export const ProjectPropertiesDialog = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        withDialog(PROJECT_PROPERTIES_DIALOG_NAME)(
            ({ classes, open, closeDialog, handleDelete, project }: ProjectPropertiesDialogProps) =>
                <Dialog open={open}
                    onClose={closeDialog}
                    fullWidth
                    maxWidth='sm'>
                    <DialogTitle>Properties</DialogTitle>
                    <DialogContent>
                        <ProjectPropertiesForm />
                        {project && project.properties &&
                            Object.keys(project.properties).map(k =>
                                Array.isArray(project.properties[k])
                                    ? project.properties[k].map((v: string) =>
                                        getPropertyChip(
                                            k, v,
                                            handleDelete(k, v),
                                            classes.tag))
                                    : getPropertyChip(
                                        k, project.properties[k],
                                        handleDelete(k, project.properties[k]),
                                        classes.tag)
                            )
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
        )
    ));
