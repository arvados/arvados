import { Redirect, RouteProps } from "react-router";
import * as React from "react";
import { connect } from "react-redux";
import authActions from "../../store/auth-action";

interface ApiTokenProps {
    saveApiToken: (token: string) => void;
}

class ApiToken extends React.Component<ApiTokenProps & RouteProps, {}> {
    static getUrlParameter(search: string, name: string) {
        const safeName = name.replace(/[\[]/, '\\[').replace(/[\]]/, '\\]');
        const regex = new RegExp('[\\?&]' + safeName + '=([^&#]*)');
        const results = regex.exec(search);
        return results === null ? '' : decodeURIComponent(results[1].replace(/\+/g, ' '));
    };

    componentDidMount() {
        const search = this.props.location ? this.props.location.search : "";
        const apiToken = ApiToken.getUrlParameter(search, 'api_token');
        this.props.saveApiToken(apiToken);
    }
    render() {
        return <Redirect to="/"/>
    }
}

export default connect<ApiTokenProps>(null, {
    saveApiToken: authActions.saveApiToken
})(ApiToken);
