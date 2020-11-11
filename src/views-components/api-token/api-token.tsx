// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RouteProps } from "react-router";
import * as React from "react";
import { connect, DispatchProp } from "react-redux";
import { saveApiToken } from "~/store/auth/auth-action";
import { getUrlParameter } from "~/common/url";
import { AuthService } from "~/services/auth-service/auth-service";
import { navigateToRootProject, navigateToLinkAccount } from "~/store/navigation/navigation-action";
import { Config } from "~/common/config";
import { getAccountLinkData } from "~/store/link-account-panel/link-account-panel-actions";

interface ApiTokenProps {
    authService: AuthService;
    config: Config;
    loadMainApp: boolean;
}

export const ApiToken = connect()(
    class extends React.Component<ApiTokenProps & RouteProps & DispatchProp<any>, {}> {
        componentDidMount() {
            const search = this.props.location ? this.props.location.search : "";
            const apiToken = getUrlParameter(search, 'api_token');
            const loadMainApp = this.props.loadMainApp;
            this.props.dispatch<any>(saveApiToken(apiToken)).finally(() => {
                const redirectURL = this.props.authService.getTargetURL();

                if (redirectURL) {
                    this.props.authService.removeTargetURL();
                    window.location.href = redirectURL;
                }
                else if (loadMainApp) {
                    if (this.props.dispatch(getAccountLinkData())) {
                        this.props.dispatch(navigateToLinkAccount);
                    }
                    else {
                        this.props.dispatch(navigateToRootProject);
                    }
                }
            });
        }
        render() {
            return <div />;
        }
    }
);
