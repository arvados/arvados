// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import {
    ApiClientAuthorizationPanelRoot,
    ApiClientAuthorizationPanelRootDataProps,
    ApiClientAuthorizationPanelRootActionProps
} from '~/views/api-client-authorization-panel/api-client-authorization-panel-root';
import { openApiClientAuthorizationContextMenu } from '~/store/context-menu/context-menu-actions';

const mapStateToProps = (state: RootState): ApiClientAuthorizationPanelRootDataProps => {
    return {
        apiClientAuthorizations: state.apiClientAuthorizations,
        hasApiClientAuthorizations: state.apiClientAuthorizations.length > 0
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ApiClientAuthorizationPanelRootActionProps => ({
    openRowOptions: (event, apiClientAuthorization) => {
        dispatch<any>(openApiClientAuthorizationContextMenu(event, apiClientAuthorization));
    }
});

export const ApiClientAuthorizationPanel = connect(mapStateToProps, mapDispatchToProps)(ApiClientAuthorizationPanelRoot);