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
    handleDelete: (key: string) => void;
}

const mapStateToProps = ({ detailsPanel, resources }: RootState): ProjectPropertiesDialogDataProps => {
    const project = getResource(detailsPanel.resourceUuid)(resources) as ProjectResource;
    return { project };
};

const mapDispatchToProps = (dispatch: Dispatch): ProjectPropertiesDialogActionProps => ({
    handleDelete: (key: string) => dispatch<any>(deleteProjectProperty(key))
});

type ProjectPropertiesDialogProps =  ProjectPropertiesDialogDataProps & ProjectPropertiesDialogActionProps & WithDialogProps<{}> & WithStyles<CssRules>;

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
                        Object.keys(project.properties).map(k => {
                            return <Chip key={k} className={classes.tag}
                                onDelete={() => handleDelete(k)}
                                label={`${k}: ${project.properties[k]}`} />;
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