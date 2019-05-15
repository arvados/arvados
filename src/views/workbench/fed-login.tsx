// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { AuthState } from '~/store/auth/auth-reducer';
import { User } from "~/models/user";
import { getSaltedToken } from '~/store/auth/auth-action-session';
import { Config } from '~/common/config';

export interface FedLoginProps {
    user?: User;
    apiToken?: string;
    homeCluster: string;
    remoteHostsConfig: { [key: string]: Config };
}

const mapStateToProps = ({ auth }: RootState) => ({
    user: auth.user,
    apiToken: auth.apiToken,
    remoteHostsConfig: auth.remoteHostsConfig,
    homeCluster: auth.homeCluster,
});

export const FedLogin = connect(mapStateToProps)(
    class extends React.Component<FedLoginProps> {
        render() {
            const { apiToken, user, homeCluster, remoteHostsConfig } = this.props;
            if (!apiToken || !user || !user.uuid.startsWith(homeCluster)) {
                return <></>;
            }
            const [, tokenUuid, token] = apiToken.split("/");
            return <div id={"fedtoken-iframe-div"}>
                {Object.keys(remoteHostsConfig)
                    .map((k) => k !== homeCluster &&
                        <iframe key={k} src={`${remoteHostsConfig[k].workbench2Url}/fedtoken?api_token=${getSaltedToken(k, tokenUuid, token)}`} style={{
                            height: 0,
                            width: 0,
                            visibility: "hidden"
                        }}
                        />)}
            </div>;
        }
    });
