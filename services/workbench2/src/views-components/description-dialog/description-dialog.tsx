// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dialog, DialogContent, DialogActions, Button, Typography } from "@mui/material";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { Dispatch, compose } from "redux";
import descriptionDialogActions, { DESCRIPTION_DIALOG, DescriptionDialogData } from "store/description-dialog/description-dialog-actions";
import { WithDialogProps, withDialog } from "store/dialog/with-dialog";
import { getResource } from "store/resources/resources";
import { getDialog } from "store/dialog/dialog-reducer";
import { ProjectResource } from "models/project";
import { CollectionResource } from "models/collection";
import { WorkflowResource } from "models/workflow";
import { ContainerRequestResource } from "models/container-request";

type DescribedResource = ProjectResource | CollectionResource | WorkflowResource | ContainerRequestResource

interface DescriptionDialogDataProps {
    description: string,
}

const mapStateToProps = (state: RootState): DescriptionDialogDataProps => {
    const dialog = getDialog<DescriptionDialogData>(state.dialog, DESCRIPTION_DIALOG);
    const resource = getResource<DescribedResource>(dialog?.data.uuid)(state.resources);
    return {
        description: resource?.description || "",
    }
};

interface DescriptionDialogActionProps {
    closeDialog: Function;
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    closeDialog: () => dispatch<any>(descriptionDialogActions.closeDialog()),
});

type DescriptionDialogComponentProps = DescriptionDialogDataProps & DescriptionDialogActionProps & WithDialogProps<DescriptionDialogData>;

export const DescriptionDialogComponent = (props: DescriptionDialogComponentProps) => {
    const { open, description } = props;

    return (
        <Dialog
            open={open}
            onClose={props.closeDialog}
            maxWidth="lg"
            fullWidth={true}
        >
            <DialogContent>
                <Typography
                    component="div"
                    //dangerouslySetInnerHTML is ok here only if description is sanitized,
                    //which it is before it is loaded into the redux store
                    dangerouslySetInnerHTML={{
                        __html: description,
                    }}
                />
            </DialogContent>
            <DialogActions style={{ margin: "0px 12px 12px" }}>
                <Button
                    data-cy="confirmation-dialog-ok-btn"
                    variant="contained"
                    color="primary"
                    type="submit"
                    onClick={props.closeDialog}
                >
                    Close
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export const DescriptionDialog = compose(
    withDialog(DESCRIPTION_DIALOG),
    connect(mapStateToProps, mapDispatchToProps)
)(DescriptionDialogComponent);
