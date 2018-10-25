// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RouteProps } from "react-router";
import * as React from "react";
import { connect, DispatchProp } from "react-redux";
import { getUserDetails, saveApiToken } from "~/store/auth/auth-action";
import { getUrlParameter } from "~/common/url";
import { AuthService } from "~/services/auth-service/auth-service";
import { navigateToRootProject } from "~/store/navigation/navigation-action";

interface ApiTokenProps {
    authService: AuthService;
}

export const ApiToken = connect()(
    class extends React.Component<ApiTokenProps & RouteProps & DispatchProp<any>, {}> {
        componentDidMount() {
            const search = this.props.location ? this.props.location.search : "";
            const apiToken = getUrlParameter(search, 'api_token');
            this.props.dispatch(saveApiToken(apiToken));
            this.props.dispatch<any>(getUserDetails()).finally(() => {
                this.props.dispatch(navigateToRootProject);
            });
        }
        render() {
            return <div/>;
        }
    }
);
