// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { } from '~/store/keep-services/keep-services-actions';
import { 
    KeepServicePanelRoot, 
    KeepServicePanelRootDataProps, 
    KeepServicePanelRootActionProps 
} from '~/views/keep-service-panel/keep-service-panel-root';
import { openKeepServiceContextMenu } from '~/store/context-menu/context-menu-actions';

const mapStateToProps = (state: RootState): KeepServicePanelRootDataProps => {
    return {
        keepServices: state.keepServices,
        hasKeepSerices: state.keepServices.length > 0
    };
};

const mapDispatchToProps = (dispatch: Dispatch): KeepServicePanelRootActionProps => ({
    openRowOptions: (event, index, keepService) => {
        dispatch<any>(openKeepServiceContextMenu(event, index, keepService));
    }
});

export const KeepServicePanel = connect(mapStateToProps, mapDispatchToProps)(KeepServicePanelRoot);