// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RouteProps } from "react-router";
import React from "react";
import { RootState } from "store/store";
import { connect, DispatchProp } from "react-redux";
import { saveApiToken } from "store/auth/auth-action";
import { getUrlParameter } from "common/url";
import { AuthService } from "services/auth-service/auth-service";
import { navigateToRootProject, navigateToLinkAccount } from "store/navigation/navigation-action";
import { Config } from "common/config";
import { getAccountLinkData } from "store/link-account-panel/link-account-panel-actions";
import { replace } from "react-router-redux";
import { User } from "models/user";

interface ApiTokenProps {
    authService: AuthService;
    config: Config;
    loadMainApp: boolean;
    user?: User;
}

export const ApiToken = connect((state: RootState) => ({
    user: state.auth.user,
}), null)(
    class extends React.Component<ApiTokenProps & RouteProps & DispatchProp<any>, {}> {
        componentDidMount() {
            const search = this.props.location ? this.props.location.search : "";
            const apiToken = getUrlParameter(search, 'api_token');
            this.props.dispatch<any>(saveApiToken(apiToken));
        }

        componentDidUpdate() {
            const redirectURL = this.props.authService.getTargetURL();

            if (this.props.loadMainApp && this.props.user) {
                if (redirectURL) {
                    this.props.authService.removeTargetURL();
                    this.props.dispatch(replace(redirectURL));
                }
                else if (this.props.dispatch(getAccountLinkData())) {
                    this.props.dispatch(navigateToLinkAccount);
                }
                else {
                    this.props.dispatch(navigateToRootProject);
                }
            }
        }

        render() {
            return <div />;
        }
    }
);
