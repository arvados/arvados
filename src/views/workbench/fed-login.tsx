// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { AuthState } from '~/store/auth/auth-reducer';
import { getSaltedToken } from '~/store/auth/auth-action-session';

export interface FedLoginProps {
    auth: AuthState;
}

const mapStateToProps = ({ auth }: RootState) => ({ auth });

export const FedLogin = connect(mapStateToProps)(
    class extends React.Component<FedLoginProps> {
        render() {
            const auth = this.props.auth;
            const remoteHostsConfig = auth.remoteHostsConfig;
            if (!auth.user || !auth.user.uuid.startsWith(auth.homeCluster)) {
                return <></>;
            }
            return <div>
                {Object.keys(remoteHostsConfig)
                    .filter((k) => k !== auth.homeCluster)
                    .map((k) => <iframe key={k} src={"https://" + remoteHostsConfig[k].workbench2Url} style={{ visibility: "hidden" }} />)}
            </div>;
        }
    });
