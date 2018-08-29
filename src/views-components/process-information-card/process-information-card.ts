// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { openProcessContextMenu } from '~/store/context-menu/context-menu-actions';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { ProcessInformationCard as InformationCardComponent, ProcessInformationCardDataProps } from '~/views/process-panel/process-information-card';

type InformationCardActionProps = Pick<ProcessInformationCardDataProps, 'onContextMenu'>;

const mapStateToProps = (state: RootState) => ({
    // todo: change for processPanel
    item: state.collectionPanel.item
});

const mapDispatchToProps = (dispatch: Dispatch): InformationCardActionProps => ({
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => {
        dispatch<any>(openProcessContextMenu(event));
    }
});

export const ProcessInformationCard = connect(mapStateToProps, mapDispatchToProps)(InformationCardComponent);