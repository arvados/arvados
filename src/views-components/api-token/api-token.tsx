// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Redirect, RouteProps } from "react-router";
import * as React from "react";
import { connect, DispatchProp } from "react-redux";
import { authActions, getUserDetails } from "../../store/auth/auth-action";
import { authService } from "../../services/services";
import { getProjectList } from "../../store/project/project-action";
import { getUrlParameter } from "../../common/url";

interface ApiTokenProps {
}

export const ApiToken = connect()(
    class extends React.Component<ApiTokenProps & RouteProps & DispatchProp<any>, {}> {
        componentDidMount() {
            const search = this.props.location ? this.props.location.search : "";
            const apiToken = getUrlParameter(search, 'api_token');
            this.props.dispatch(authActions.SAVE_API_TOKEN(apiToken));
            this.props.dispatch<any>(getUserDetails()).then(() => {
                const rootUuid = authService.getRootUuid();
                this.props.dispatch(getProjectList(rootUuid));
            });
        }
        render() {
            return <Redirect to="/"/>;
        }
    }
);
