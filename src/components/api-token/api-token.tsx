// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Redirect, RouteProps } from "react-router";
import * as React from "react";
import { connect, DispatchProp } from "react-redux";
import authActions, { getUserDetails } from "../../store/auth/auth-action";

interface ApiTokenProps {
}

class ApiToken extends React.Component<ApiTokenProps & RouteProps & DispatchProp<any>, {}> {
    static getUrlParameter(search: string, name: string) {
        const safeName = name.replace(/[\[]/, '\\[').replace(/[\]]/, '\\]');
        const regex = new RegExp('[\\?&]' + safeName + '=([^&#]*)');
        const results = regex.exec(search);
        return results === null ? '' : decodeURIComponent(results[1].replace(/\+/g, ' '));
    };

    componentDidMount() {
        const search = this.props.location ? this.props.location.search : "";
        const apiToken = ApiToken.getUrlParameter(search, 'api_token');
        this.props.dispatch(authActions.SAVE_API_TOKEN(apiToken));
        this.props.dispatch(getUserDetails());
    }
    render() {
        return <Redirect to="/"/>
    }
}

export default connect()(ApiToken);
