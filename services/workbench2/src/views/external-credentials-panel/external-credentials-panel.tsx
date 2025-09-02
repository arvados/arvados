// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { camelCase } from "lodash";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from "react-router";
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { ArvadosTheme } from "common/custom-theme";
import { EXTERNAL_CREDENTIALS_PANEL } from "store/external-credentials/external-credentials-actions";
import {
    ResourceName,
    ResourceExpiresAtDate,
    RenderResourceStringField,
    RenderScopes,
} from "views-components/data-explorer/renderers";
import { ProcessIcon } from "components/icon/icon";
import { openProcessContextMenu } from "store/context-menu/context-menu-actions";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { navigateTo } from "store/navigation/navigation-action";
import { RootState } from "store/store";
import { createTree } from "models/tree";
import { getProcess } from "store/processes/process";
import { ResourcesState } from "store/resources/resources";
import { toggleOne } from "store/multiselect/multiselect-actions";
import { ExternalCredential } from "models/external-credential";

type CssRules = "toolbar" | "button" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing(3),
        textAlign: "right",
    },
    button: {
        marginLeft: theme.spacing(1),
    },
    root: {
        width: "100%",
        boxShadow: "0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)",
    },
});

export enum ExternalCredentialsPanelColumnNames {
    NAME = "Name",
    DESCRIPTION = "Description",
    EXTERNAL_ID = "External ID",
    CREDENTIAL_CLASS = "Credential class",
    EXPIRES_AT = "Expires at",
    SCOPES = "Scopes",
}

export const externalCredentialsPanelColumns: DataColumns<string, ExternalCredential> = [
    {
        name: ExternalCredentialsPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.DESCRIPTION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid =>
            <RenderResourceStringField<ExternalCredential>
                uuid={uuid}
                field={camelCase(ExternalCredentialsPanelColumnNames.DESCRIPTION) as keyof ExternalCredential} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.CREDENTIAL_CLASS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid =>
            <RenderResourceStringField<ExternalCredential>
                uuid={uuid}
                field={camelCase(ExternalCredentialsPanelColumnNames.CREDENTIAL_CLASS) as keyof ExternalCredential} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.EXTERNAL_ID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid =>
            <RenderResourceStringField<ExternalCredential>
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

interface ExternalCredentialsPanelDataProps {
    resources: ResourcesState;
}

interface ExternalCredentialsPanelActionProps {
    onItemClick: (item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = (state: RootState): ExternalCredentialsPanelDataProps => ({
    resources: state.resources,
});

type ExternalCredentialsPanelProps = ExternalCredentialsPanelDataProps &
    ExternalCredentialsPanelActionProps &
    DispatchProp &
    WithStyles<CssRules> &
    RouteComponentProps<{ id: string }>;

export const ExternalCredentialsPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<ExternalCredentialsPanelProps> {
            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const process = getProcess(resourceUuid)(this.props.resources);
                if (process) {
                    this.props.dispatch<any>(openProcessContextMenu(event, process));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            };

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            };

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(toggleOne(uuid))
            };

            render() {
                return (
                    <div className={this.props.classes.root}>
                        <DataExplorer
                            id={EXTERNAL_CREDENTIALS_PANEL}
                            onRowClick={this.handleRowClick}
                            onRowDoubleClick={this.handleRowDoubleClick}
                            onContextMenu={this.handleContextMenu}
                            contextMenuColumn={false}
                            defaultViewIcon={ProcessIcon}
                            defaultViewMessages={["External credentials list empty."]}
                        />
                    </div>
                );
            }
        }
    )
);
