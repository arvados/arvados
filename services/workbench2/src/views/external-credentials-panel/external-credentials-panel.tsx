// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { Grid, Button } from "@mui/material";
import { camelCase, noop } from "lodash";
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from "react-router";
import { Dispatch } from "redux";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { ArvadosTheme } from "common/custom-theme";
import { EXTERNAL_CREDENTIALS_PANEL, openNewExternalCredentialDialog } from "store/external-credentials/external-credentials-actions";
import {
    ResourceNameNoLink,
    ResourceExpiresAtDate,
    RenderResourceStringField,
    RenderScopes,
    RenderDescription,
} from "views-components/data-explorer/renderers";
import { FolderKeyIcon, AddIcon } from "components/icon/icon";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { RootState } from "store/store";
import { createTree } from "models/tree";
import { ResourcesState, getResource } from "store/resources/resources";
import { toggleOne } from "store/multiselect/multiselect-actions";
import { ExternalCredential } from "models/external-credential";
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { openContextMenuAndSelect } from "store/context-menu/context-menu-actions";

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
        render: uuid => <ResourceNameNoLink uuid={uuid} />,
    },
    {
        name: ExternalCredentialsPanelColumnNames.DESCRIPTION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <RenderDescription uuid={uuid} />,
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
    onNewCredential: () => void;
    openContextMenuAndSelect: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource) => void;
    loadDetailsPanel: (resourceUuid: string) => void;
    toggleOne: (uuid: string) => void;
}
const mapStateToProps = (state: RootState): ExternalCredentialsPanelDataProps => ({
    resources: state.resources,
});

const mapDispatchToProps = (dispatch: Dispatch): ExternalCredentialsPanelActionProps => ({
    onNewCredential: () => dispatch<any>(openNewExternalCredentialDialog()),
    openContextMenuAndSelect: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource) => dispatch<any>(openContextMenuAndSelect(event, resource)),
    loadDetailsPanel: (resourceUuid: string) => dispatch<any>(loadDetailsPanel(resourceUuid)),
    toggleOne: (uuid: string) => dispatch<any>(toggleOne(uuid)),
});

type ExternalCredentialsPanelProps = ExternalCredentialsPanelDataProps &
    ExternalCredentialsPanelActionProps &
    DispatchProp &
    WithStyles<CssRules> &
    RouteComponentProps<{ id: string }>;

export const ExternalCredentialsPanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<ExternalCredentialsPanelProps> {
            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const externalCredential = getResource<ExternalCredential>(resourceUuid)(this.props.resources);
                if (externalCredential) {
                    this.props.openContextMenuAndSelect(event, {
                        name: externalCredential.name,
                        uuid: externalCredential.uuid,
                        ownerUuid: externalCredential.ownerUuid,
                        kind: externalCredential.kind,
                        menuKind: externalCredential.kind
                    });
                }
                this.props.loadDetailsPanel(resourceUuid);
            };

            handleRowClick = (uuid: string) => {
                this.props.toggleOne(uuid);
            };

            render() {
                return (
                    <div className={this.props.classes.root}>
                        <DataExplorer
                            id={EXTERNAL_CREDENTIALS_PANEL}
                            onRowClick={this.handleRowClick}
                            onRowDoubleClick={noop}
                            onContextMenu={this.handleContextMenu}
                            contextMenuColumn={false}
                            defaultViewIcon={FolderKeyIcon}
                            defaultViewMessages={["External credentials list empty."]}
                            hideColumnSelector
                            actions={
                                <Grid container justifyContent='flex-end'>
                                    <Button
                                        data-cy="groups-panel-new-group"
                                        variant="contained"
                                        color="primary"
                                        onClick={this.props.onNewCredential}>
                                        <AddIcon /> New External Credential
                                    </Button>
                                </Grid>
                            }
                        />
                    </div>
                );
            }
        }
    )
);
