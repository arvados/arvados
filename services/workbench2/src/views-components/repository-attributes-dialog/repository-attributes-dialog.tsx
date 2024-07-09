// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography, Grid } from "@material-ui/core";
import { WithDialogProps } from "store/dialog/with-dialog";
import { withDialog } from 'store/dialog/with-dialog';
import { REPOSITORY_ATTRIBUTES_DIALOG } from "store/repositories/repositories-actions";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { compose } from "redux";
import { RepositoryResource } from "models/repositories";

type CssRules = 'rightContainer' | 'leftContainer' | 'spacing';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    rightContainer: {
        textAlign: 'right',
        paddingRight: theme.spacing.unit * 2,
        color: theme.palette.grey["500"]
    },
    leftContainer: {
        textAlign: 'left',
        paddingLeft: theme.spacing.unit * 2
    },
    spacing: {
        paddingTop: theme.spacing.unit * 2
    },
});

interface RepositoryAttributesDataProps {
    repositoryData: RepositoryResource;
}

type RepositoryAttributesProps = RepositoryAttributesDataProps & WithStyles<CssRules>;

export const RepositoryAttributesDialog = compose(
    withDialog(REPOSITORY_ATTRIBUTES_DIALOG),
    withStyles(styles))(
        (props: WithDialogProps<RepositoryAttributesProps> & RepositoryAttributesProps) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth="sm">
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    <Typography variant='body1' className={props.classes.spacing}>
                        {props.data.repositoryData && attributes(props.data.repositoryData, props.classes)}
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={props.closeDialog}>
                        Close
                </Button>
                </DialogActions>
            </Dialog>
    );

const attributes = (repositoryData: RepositoryResource, classes: any) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name } = repositoryData;
    return (
        <span>
            <Grid container direction="row">
                <Grid item xs={5} className={classes.rightContainer}>
                    <Grid item>Name</Grid>
                    <Grid item>Owner uuid</Grid>
                    <Grid item>Created at</Grid>
                    <Grid item>Modified at</Grid>
                    <Grid item>Modified by user uuid</Grid>
                    <Grid item>Modified by client uuid</Grid>
                    <Grid item>uuid</Grid>
                </Grid>
                <Grid item xs={7} className={classes.leftContainer}>
                    <Grid item>{name}</Grid>
                    <Grid item>{ownerUuid}</Grid>
                    <Grid item>{createdAt}</Grid>
                    <Grid item>{modifiedAt}</Grid>
                    <Grid item>{modifiedByUserUuid}</Grid>
                    <Grid item>{modifiedByClientUuid}</Grid>
                    <Grid item>{uuid}</Grid>
                </Grid>
            </Grid>
        </span>
    );
};
