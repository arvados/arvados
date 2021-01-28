// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { connect } from 'react-redux';
import { fetchOwnerNameByUuid } from '~/store/owner-name-uuid-enhancer/owner-name-uuid-enhancer-actions';

export interface OwnerNameUuidEnhancerOwnProps {
    uuid: string;
}

export interface OwnerNameUuidEnhancerRootDataProps {
    ownerNamesMap: any;
}

export interface OwnerNameUuidEnhancerDispatchProps {
    fetchOwner: Function;
}

export type OwnerNameUuidEnhancerProps = OwnerNameUuidEnhancerOwnProps & OwnerNameUuidEnhancerRootDataProps & OwnerNameUuidEnhancerDispatchProps;

export const OwnerNameUuidEnhancer = ({ uuid, ownerNamesMap, fetchOwner }: OwnerNameUuidEnhancerProps) => {
    React.useEffect(() => {
        if (!ownerNamesMap[uuid]) {
            fetchOwner(uuid);
        }
    }, [uuid, ownerNamesMap, fetchOwner]);

    return <span>{uuid}{ownerNamesMap[uuid] ? ` (${ownerNamesMap[uuid]})` : ''}</span>;
};

const mapStateToProps = (state: RootState): OwnerNameUuidEnhancerRootDataProps => {
    return {
        ownerNamesMap: state.ownerNameUuidEnhancer,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): OwnerNameUuidEnhancerDispatchProps => ({
    fetchOwner: (uuid: string) => dispatch<any>(fetchOwnerNameByUuid(uuid))
});

export default connect<OwnerNameUuidEnhancerRootDataProps, OwnerNameUuidEnhancerDispatchProps, OwnerNameUuidEnhancerOwnProps>(mapStateToProps, mapDispatchToProps)
    (OwnerNameUuidEnhancer);
