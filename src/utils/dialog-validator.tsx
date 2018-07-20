// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

type ValidatorProps = {
    value: string,
    render: (hasError: boolean) => React.ReactElement<any>;
    isUniqName?: boolean;
    validators: Array<(value: string) => string>;
};

class Validator extends React.Component<ValidatorProps & WithStyles<CssRules>> {
    render() {
        const { classes, value, isUniqName } = this.props;

        return (
            <span>
                {this.props.render(!this.isValid(value))}
                {isUniqName ? <span className={classes.formInputError}>Project with this name already exists</span> : null}
                {this.props.validators.map(validate => {
                    const errorMsg = validate(value);
                    return errorMsg ? <span className={classes.formInputError}>{errorMsg}</span> : null;
                })}
            </span>
        );
    }

    isValid(value: string) {
        return this.props.validators.every(validate => validate(value).length === 0);
    }
}

export const required = (value: string) => value.length > 0 ? "" : "This value is required";
export const maxLength = (max: number) => (value: string) => value.length <= max ? "" : `This field should have max ${max} characters.`;
export const isUniq = (getError: () => string) => (value: string) => getError() ? "Project with this name already exists" : "";

type CssRules = "formInputError";

const styles: StyleRulesCallback<CssRules> = theme => ({
    formInputError: {
        color: "#ff0000",
        marginLeft: "5px",
        fontSize: "11px",
    }
});

export default withStyles(styles)(Validator);