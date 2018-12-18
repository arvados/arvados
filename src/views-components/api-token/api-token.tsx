// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RouteProps } from "react-router";
import * as React from "react";
import { connect, DispatchProp } from "react-redux";
import { authActions, getUserDetails, saveApiToken } from "~/store/auth/auth-action";
import { getUrlParameter } from "~/common/url";
import { AuthService } from "~/services/auth-service/auth-service";
import { navigateToRootProject } from "~/store/navigation/navigation-action";
import { User } from "~/models/user";
import { Config } from "~/common/config";
import { initSessions } from "~/store/auth/auth-action-session";

interface ApiTokenProps {
    authService: AuthService;
    config: Config;
}

export const ApiToken = connect()(
    class extends React.Component<ApiTokenProps & RouteProps & DispatchProp<any>, {}> {
        componentDidMount() {
            const search = this.props.location ? this.props.location.search : "";
            const apiToken = getUrlParameter(search, 'api_token');
            this.props.dispatch(saveApiToken(apiToken));
            this.props.dispatch<any>(getUserDetails()).then((user: User) => {
                this.props.dispatch(initSessions(this.props.authService, this.props.config, user));
            }).finally(() => {
                this.props.dispatch(navigateToRootProject);
            });
        }
        render() {
            return <div/>;
        }
    }
);
