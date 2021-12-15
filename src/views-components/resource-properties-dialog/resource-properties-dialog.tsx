// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from 'store/store';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { RESOURCE_PROPERTIES_DIALOG_NAME } from 'store/details-panel/details-panel-action';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    withStyles,
    StyleRulesCallback,
    WithStyles
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { ResourcePropertiesDialogForm } from 'views-components/resource-properties-dialog/resource-properties-dialog-form';
import { getResource } from 'store/resources/resources';
import { getPropertyChip } from "../resource-properties-form/property-chip";
import { deleteResourceProperty } from "store/resources/resources-actions";
import { ResourceWithProperties } from "models/resource";

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface ResourcePropertiesDialogDataProps {
    resource: ResourceWithProperties;
}

interface ResourcePropertiesDialogActionProps {
    handleDelete: (uuid: string, key: string, value: string) => void;
}

const mapStateToProps = ({ detailsPanel, resources, properties }: RootState): ResourcePropertiesDialogDataProps => ({
    resource: getResource(detailsPanel.resourceUuid)(resources) as ResourceWithProperties,
});

const mapDispatchToProps = (dispatch: Dispatch): ResourcePropertiesDialogActionProps => ({
    handleDelete: (uuid: string, key: string, value: string) => () => dispatch<any>(deleteResourceProperty(uuid, key, value)),
});

type ResourcePropertiesDialogProps = ResourcePropertiesDialogDataProps & ResourcePropertiesDialogActionProps & WithDialogProps<{}> & WithStyles<CssRules>;

export const ResourcePropertiesDialog = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        withDialog(RESOURCE_PROPERTIES_DIALOG_NAME)(
            ({ classes, open, closeDialog, handleDelete, resource }: ResourcePropertiesDialogProps) =>
                <Dialog open={open}
                    onClose={closeDialog}
                    fullWidth
                    maxWidth='sm'>
                    <div data-cy='resource-properties-dialog'>
                    <DialogTitle>Edit properties</DialogTitle>
                    <DialogContent>
                        <ResourcePropertiesDialogForm uuid={resource ? resource.uuid : ''} />
                        {resource && resource.properties &&
                            Object.keys(resource.properties).map(k =>
                                Array.isArray(resource.properties[k])
                                    ? resource.properties[k].map((v: string) =>
                                        getPropertyChip(
                                            k, v,
                                            handleDelete(resource.uuid, k, v),
                                            classes.tag))
                                    : getPropertyChip(
                                        k, resource.properties[k],
                                        handleDelete(resource.uuid, k, resource.properties[k]),
                                        classes.tag)
                            )
                        }
                    </DialogContent>
                    <DialogActions>
                        <Button
                            data-cy='close-dialog-btn'
                            variant='text'
                            color='primary'
                            onClick={closeDialog}>
                            Close
                    </Button>
                    </DialogActions>
                    </div>
                </Dialog>
            )
    ));
