// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { camelCase } from "lodash";
import { ExternalCredential } from "models/external-credential";
import { createTree } from "models/tree";
import {
    ResourceNameNoLink,
    ResourceExpiresAtDate,
    RenderResourceStringField,
    RenderScopes,
    RenderDescriptionInTD,
} from "views-components/data-explorer/renderers";

export enum ExternalCredentialsPanelColumnNames {
    NAME = "Name",
    DESCRIPTION = "Description",
    EXTERNAL_ID = "External ID",
    CREDENTIAL_CLASS = "Credential class",
    EXPIRES_AT = "Expires at",
    SCOPES = "Scopes"
}

export const externalCredentialsPanelColumns: DataColumns<string, ExternalCredential> = [
    {
        name: ExternalCredentialsPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceNameNoLink uuid={uuid} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.DESCRIPTION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <RenderDescriptionInTD uuid={uuid} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.CREDENTIAL_CLASS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <RenderResourceStringField<ExternalCredential>
            uuid={uuid}
            field={camelCase(ExternalCredentialsPanelColumnNames.CREDENTIAL_CLASS) as keyof ExternalCredential} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.EXTERNAL_ID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <RenderResourceStringField<ExternalCredential>
            uuid={uuid}
            field={camelCase(ExternalCredentialsPanelColumnNames.EXTERNAL_ID) as keyof ExternalCredential} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.EXPIRES_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceExpiresAtDate uuid={uuid} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.SCOPES,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <RenderScopes uuid={uuid} />,
    },
];
