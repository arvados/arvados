// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { User } from "~/models/user";
import { getSaltedToken } from '~/store/auth/auth-action-session';
import { Config } from '~/common/config';

export interface FedLoginProps {
    user?: User;
    apiToken?: string;
    localCluster: string;
    remoteHostsConfig: { [key: string]: Config };
}

const mapStateToProps = ({ auth }: RootState) => ({
    user: auth.user,
    apiToken: auth.apiToken,
    remoteHostsConfig: auth.remoteHostsConfig,
    localCluster: auth.localCluster,
});

export const FedLogin = connect(mapStateToProps)(
    class extends React.Component<FedLoginProps> {
        render() {
            const { apiToken, user, localCluster, remoteHostsConfig } = this.props;
            if (!apiToken || !user || !user.uuid.startsWith(localCluster)) {
                return <></>;
            }
            const [, tokenUuid, token] = apiToken.split("/");
            return <div id={"fedtoken-iframe-div"}>
                {Object.keys(remoteHostsConfig)
                    .map((k) => {
                        if (k === localCluster) {
                            return;
                        }
                        if (!remoteHostsConfig[k].workbench2Url) {
                            console.log(`Cluster ${k} does not define workbench2Url.  Federated login / cross-site linking to ${k} is unavailable.  Tell the admin of ${k} to set Services->Workbench2->ExternalURL in config.yml.`);
                            return;
                        }
                        return <iframe key={k} src={`${remoteHostsConfig[k].workbench2Url}/fedtoken?api_token=${getSaltedToken(k, tokenUuid, token)}`} style={{
                            height: 0,
                            width: 0,
                            visibility: "hidden"
                        }}
                        />;
                    })}
            </div>;
        }
    });
