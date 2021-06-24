// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ErrorIcon } from "components/icon/icon";
import { Tooltip } from "@material-ui/core";
import { disallowSlash } from "validators/valid-name";
import { connect } from "react-redux";
import { RootState } from "store/store";

interface WarningComponentProps {
    text: string;
    rules: RegExp[];
    message: string;
}

export const WarningComponent = ({ text, rules, message }: WarningComponentProps) =>
    !text ? <Tooltip title={"No name"}><ErrorIcon /></Tooltip>
        : (rules.find(aRule => text.match(aRule) !== null)
            ? message
                ? <Tooltip title={message}><ErrorIcon /></Tooltip>
                : <ErrorIcon />
            : null);

interface IllegalNamingWarningProps {
    name: string;
    validate: RegExp[];
}


export const IllegalNamingWarning = connect(
    (state: RootState) => {
        return {
            validate: (state.auth.config.clusterConfig.Collections.ForwardSlashNameSubstitution === "" ?
                [disallowSlash] : [])
        };
    })(({ name, validate }: IllegalNamingWarningProps) =>
        <WarningComponent
            text={name} rules={validate}
            message="Names embedding '/' will be renamed or invisible to file system access (arv-mount or WebDAV)" />);
