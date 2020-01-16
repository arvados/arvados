// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ErrorIcon } from "~/components/icon/icon";
import { disallowSlash } from "~/validators/valid-name";
import { Tooltip } from "@material-ui/core";

interface WarningComponentProps {
    text: string;
    rules: RegExp[];
    message: string;
}

export const WarningComponent = ({ text, rules, message }: WarningComponentProps) =>
    rules.find(aRule => text.match(aRule) !== null)
        ? message
            ? <Tooltip title={message}><ErrorIcon /></Tooltip>
            : <ErrorIcon />
        : null;

interface IllegalNamingWarningProps {
    name: string;
}

export const IllegalNamingWarning = ({ name }: IllegalNamingWarningProps) =>
    <WarningComponent
        text={name} rules={[disallowSlash]}
        message="Names embedding '/' will be renamed or invisible to file system access (arv-mount or WebDAV)" />;
