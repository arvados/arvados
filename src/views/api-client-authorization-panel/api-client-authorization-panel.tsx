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
import { openApiClientAuthorizationsHelpDialog } from '~/store/api-client-authorizations/api-client-authorizations-actions';

const mapStateToProps = (state: RootState): ApiClientAuthorizationPanelRootDataProps => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ApiClientAuthorizationPanelRootActionProps => ({
    onContextMenu: (event, apiClientAuthorization) => {
        dispatch<any>(openApiClientAuthorizationContextMenu(event, apiClientAuthorization));
    },
    onItemClick: (resourceUuid: string) => { return; },
    onItemDoubleClick: uuid => { return; },
    openHelpDialog: () => {
        dispatch<any>(openApiClientAuthorizationsHelpDialog());
    }
});

export const ApiClientAuthorizationPanel = connect(mapStateToProps, mapDispatchToProps)(ApiClientAuthorizationPanelRoot);