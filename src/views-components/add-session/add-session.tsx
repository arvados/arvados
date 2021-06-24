// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RouteProps } from "react-router";
import * as React from "react";
import { connect, DispatchProp } from "react-redux";
import { getUrlParameter } from "common/url";
import { navigateToSiteManager } from "store/navigation/navigation-action";
import { addSession } from "store/auth/auth-action-session";

export const AddSession = connect()(
    class extends React.Component<RouteProps & DispatchProp<any>, {}> {
        componentDidMount() {
            const search = this.props.location ? this.props.location.search : "";
            const apiToken = getUrlParameter(search, 'api_token');
            const baseURL = getUrlParameter(search, 'baseURL');

            this.props.dispatch(addSession(baseURL, apiToken));
            this.props.dispatch(navigateToSiteManager);
        }
        render() {
            return <div />;
        }
    }
);
