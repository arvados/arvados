// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch, compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { reduxForm, startSubmit, stopSubmit, InjectedFormProps, initialize, Field, WrappedFieldProps } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { ProjectTreePicker } from '~/views-components/project-tree-picker/project-tree-picker';
import { Typography } from "@material-ui/core";
import { ResourceKind } from '~/models/resource';
import { ServiceRepository, getResourceService } from '~/services/services';
import { RootState } from '~/store/store';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/common/api/common-resource-service";
import { snackbarActions } from '../../store/snackbar/snackbar-actions';
import { require } from '~/validators/require';

export const MOVE_TO_DIALOG = 'moveToDialog';

export interface MoveToDialogResource {
    name: string;
    uuid: string;
    ownerUuid: string;
    kind: ResourceKind;
}

export const openMoveToDialog = (resource: { name: string, uuid: string, kind: ResourceKind }) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(MOVE_TO_DIALOG, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: MOVE_TO_DIALOG, data: {} }));
    };

export const moveResource = (resource: MoveToDialogResource) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const service = getResourceService(resource.kind, services);
        dispatch(startSubmit(MOVE_TO_DIALOG));
        if (service) {
            try {
                const originalResource = await service.get(resource.uuid);
                await service.update(resource.uuid, { ...originalResource, owner_uuid: resource.ownerUuid });
                dispatch(dialogActions.CLOSE_DIALOG({ id: MOVE_TO_DIALOG }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Resource has been moved', hideDuration: 2000 }));
            } catch (e) {
                const error = getCommonResourceServiceError(e);
                if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                    dispatch(stopSubmit(MOVE_TO_DIALOG, { ownerUuid: 'A resource with the same name already exists in the target project' }));
                } else {
                    dispatch(dialogActions.CLOSE_DIALOG({ id: MOVE_TO_DIALOG }));
                    dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not move the resource.', hideDuration: 2000 }));
                }
            }
        }
    };

export const MoveToProjectDialog = compose(
    withDialog(MOVE_TO_DIALOG),
    reduxForm<MoveToDialogResource>({
        form: MOVE_TO_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(moveResource(data));
        }
    })
)((props: WithDialogProps<string> & InjectedFormProps<MoveToDialogResource>) =>
    <FormDialog
        dialogTitle='Move to'
        formFields={MoveToDialogFields}
        submitLabel='Move'
        {...props}
    />);

const MoveToDialogFields = (props: InjectedFormProps<MoveToDialogResource>) =>
    <Field
        name="ownerUuid"
        component={Picker}
        validate={validation} />;

const sameUuid = (value: string, allValues: MoveToDialogResource) =>
    value === allValues.uuid && 'Cannot move the project to itself';

const validation = [require, sameUuid];

const Picker = (props: WrappedFieldProps) =>
    <div style={{ height: '144px', display: 'flex', flexDirection: 'column' }}>
        <ProjectTreePicker onChange={projectUuid => props.input.onChange(projectUuid)} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;
