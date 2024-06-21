// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { formValueSelector } from 'redux-form'
import {
    connectSharingDialog,
    saveSharingDialogChanges,
    connectSharingDialogProgress,
    SharingDialogData,
    createSharingToken,
    initializeManagementForm
} from 'store/sharing-dialog/sharing-dialog-actions';
import { WithDialogProps } from 'store/dialog/with-dialog';
import {
    SharingDialogComponent,
    SharingDialogDataProps,
    SharingDialogActionProps
} from './sharing-dialog-component';
import {
    getSharingPublicAccessFormData,
    hasChanges,
    SHARING_DIALOG_NAME,
    SHARING_MANAGEMENT_FORM_NAME,
    VisibilityLevel
} from 'store/sharing-dialog/sharing-dialog-types';
import { WithProgressStateProps } from 'store/progress-indicator/with-progress';
import { getDialog } from 'store/dialog/dialog-reducer';
import { filterResources } from 'store/resources/resources';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import { ResourceKind } from 'models/resource';

type Props = WithDialogProps<string> & WithProgressStateProps;

const sharingManagementFormSelector = formValueSelector(SHARING_MANAGEMENT_FORM_NAME);

const mapStateToProps = (state: RootState, { working, ...props }: Props): SharingDialogDataProps => {
    const dialog = getDialog<SharingDialogData>(state.dialog, SHARING_DIALOG_NAME);
    const sharedResourceUuid = dialog?.data.resourceUuid || '';
    const sharingURLsDisabled = state.auth.config.clusterConfig.Workbench.DisableSharingURLsUI;
    return ({
        ...props,
        permissions: sharingManagementFormSelector(state, 'permissions'),
        saveEnabled: hasChanges(state),
        loading: working,
        sharedResourceUuid,
        sharingURLsDisabled,
        sharingURLsNr: !sharingURLsDisabled
            ? (filterResources((resource: ApiClientAuthorization) =>
                resource.kind === ResourceKind.API_CLIENT_AUTHORIZATION &&
                resource.scopes.includes(`GET /arvados/v1/collections/${sharedResourceUuid}`) &&
                resource.scopes.includes(`GET /arvados/v1/collections/${sharedResourceUuid}/`) &&
                resource.scopes.includes('GET /arvados/v1/keep_services/accessible')
            )(state.resources) as ApiClientAuthorization[]).length
            : 0,
        privateAccess: getSharingPublicAccessFormData(state)?.visibility === VisibilityLevel.PRIVATE,
    })
};

const mapDispatchToProps = (dispatch: Dispatch, { ...props }: Props): SharingDialogActionProps => ({
    ...props,
    onClose: props.closeDialog,
    onSave: () => {
        setTimeout(() => dispatch<any>(saveSharingDialogChanges), 0);
    },
    onCreateSharingToken: (d: Date) => () => {
        dispatch<any>(createSharingToken(d));
    },
    refreshPermissions: () => {
        dispatch<any>(initializeManagementForm);
    }
});

export const SharingDialog = compose(
    connectSharingDialog,
    connectSharingDialogProgress,
    connect(mapStateToProps, mapDispatchToProps)
)(SharingDialogComponent);
