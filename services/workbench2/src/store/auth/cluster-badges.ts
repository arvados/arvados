// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { Session } from 'models/session';
import { propertiesActions } from 'store/properties/properties-actions';

export type ClusterBadge = {
    text: string,
    color: string,
    backgroundColor: string
}

export const initClusterBadges = (sessions: Session[]) => (dispatch: Dispatch) => {

    const bgColors = [
        '#2e8b57', // sea green
        '#000fd0', // royal blue
        '#fb6b1c', // orange
        '#580082', // purple
        '#733e07', // brown
        '#961e0a', // dark red
        '#ff49b4', // pink
        '#00c6c9', // turquoise
        '#c1802f', // tan
        '#1e90ff', // light blue
        '#972be2', // violet
        '#ecc700', // mustard yellow
    ];

    const badges: ClusterBadge[] = sessions.map((session, i) => {
        const color = bgColors[i % bgColors.length];
        return {
            text: session.clusterId,
            color: '#fff',
            backgroundColor: color,
        };
    });

    dispatch(propertiesActions.SET_PROPERTY({ key: 'clusterBadges', value: badges }));
};
