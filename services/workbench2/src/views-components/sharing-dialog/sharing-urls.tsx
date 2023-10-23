// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import { filterResources } from 'store/resources/resources';
import { ResourceKind } from 'models/resource';
import {
    SharingURLsComponent,
    SharingURLsComponentActionProps,
    SharingURLsComponentDataProps
} from './sharing-urls-component';
import {
    snackbarActions,
    SnackbarKind
} from 'store/snackbar/snackbar-actions';
import { deleteSharingToken } from 'store/sharing-dialog/sharing-dialog-actions';

const mapStateToProps =
    (state: RootState, ownProps: { uuid: string }): SharingURLsComponentDataProps => {
        const sharingTokens = filterResources(
            (resource: ApiClientAuthorization) =>
                resource.kind === ResourceKind.API_CLIENT_AUTHORIZATION  &&
                resource.scopes.includes(`GET /arvados/v1/collections/${ownProps.uuid}`) &&
                resource.scopes.includes(`GET /arvados/v1/collections/${ownProps.uuid}/`) &&
                resource.scopes.includes('GET /arvados/v1/keep_services/accessible')
            )(state.resources) as ApiClientAuthorization[];
        const sharingURLsPrefix = state.auth.config.keepWebInlineServiceUrl;
        return {
            collectionUuid: ownProps.uuid,
            sharingTokens,
            sharingURLsPrefix,
        }
    }

const mapDispatchToProps = (dispatch: Dispatch): SharingURLsComponentActionProps => ({
    onDeleteSharingToken(uuid: string) {
        dispatch<any>(deleteSharingToken(uuid));
    },
    onCopy(message: string) {
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message,
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS
        }));
    },
})

export const SharingURLsContent = connect(mapStateToProps, mapDispatchToProps)(SharingURLsComponent)

