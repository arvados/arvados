// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, InjectedFormProps } from "redux-form";
import { DispatchProp } from 'react-redux';
import { UserResource } from 'models/user';
import { TextField } from "components/text-field/text-field";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { NativeSelectField } from "components/select-field/select-field";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {
    CardContent,
    Button,
    Typography,
    Grid,
    InputLabel,
    Tabs,
    Tab,
    Paper,
    Tooltip,
    IconButton,
} from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { PROFILE_EMAIL_VALIDATION, PROFILE_URL_VALIDATION } from "validators/validators";
import { USER_PROFILE_PANEL_ID } from 'store/user-profile/user-profile-actions';
import { noop } from 'lodash';
import { DetailsIcon, GroupsIcon, MoreVerticalIcon } from 'components/icon/icon';
import { DataColumns } from 'components/data-table/data-column';
import { ResourceLinkHeadUuid, ResourceLinkHeadPermissionLevel, ResourceLinkHead, ResourceLinkDelete, ResourceLinkTailIsVisible, UserResourceAccountStatus } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { ResourcesState } from 'store/resources/resources';
import { DefaultView } from 'components/default-view/default-view';
import { CopyToClipboardSnackbar } from 'components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar';
import { PermissionResource } from 'models/permission';

type CssRules = 'root' | 'emptyRoot' | 'gridItem' | 'label' | 'readOnlyValue' | 'title' | 'description' | 'actions' | 'content' | 'copyIcon' | 'userProfileFormMessage';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    emptyRoot: {
        width: '100%',
        overflow: 'auto',
        padding: theme.spacing(4),
    },
    gridItem: {
        height: 45,
        marginBottom: 20
    },
    label: {
        fontSize: '0.675rem',
        color: theme.palette.grey['600']
    },
    readOnlyValue: {
        fontSize: '0.875rem',
    },
    title: {
        fontSize: '1.1rem',
    },
    description: {
        color: theme.palette.grey["600"]
    },
    actions: {
        display: 'flex',
        justifyContent: 'flex-end'
    },
    content: {
        // reserve space for the tab bar
        height: `calc(100% - ${theme.spacing(7)})`,
    },
    copyIcon: {
        marginLeft: theme.spacing(1),
        color: theme.palette.grey["500"],
        cursor: 'pointer',
        display: 'inline',
        '& svg': {
            fontSize: '1rem'
        }
    },
    userProfileFormMessage: {
        fontSize: '1.1rem',
    }
});

export interface UserProfilePanelRootActionProps {
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: UserResource | undefined) => void;
}

export interface UserProfilePanelRootDataProps {
    isAdmin: boolean;
    isSelf: boolean;
    isPristine: boolean;
    isValid: boolean;
    isInaccessible: boolean;
    userUuid: string;
    user: UserResource | undefined;
    resources: ResourcesState;
    localCluster: string;
    userProfileFormMessage: string;
}

const RoleTypes = [
    { key: '', value: '' },
    { key: 'Bio-informatician', value: 'Bio-informatician' },
    { key: 'Data Scientist', value: 'Data Scientist' },
    { key: 'Analyst', value: 'Analyst' },
    { key: 'Researcher', value: 'Researcher' },
    { key: 'Software Developer', value: 'Software Developer' },
    { key: 'System Administrator', value: 'System Administrator' },
    { key: 'Other', value: 'Other' }
];

type UserProfilePanelRootProps = InjectedFormProps<{}> & UserProfilePanelRootActionProps & UserProfilePanelRootDataProps & DispatchProp & WithStyles<CssRules>;

export enum UserProfileGroupsColumnNames {
    NAME = "Name",
    PERMISSION = "Permission",
    VISIBLE = "Visible to other members",
    UUID = "UUID",
    REMOVE = "Remove",
}

enum TABS {
    PROFILE = "PROFILE",
    GROUPS = "GROUPS",

}

export const userProfileGroupsColumns: DataColumns<PermissionResource> = [
    {
        name: UserProfileGroupsColumnNames.NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: resource => <ResourceLinkHead resource={resource as PermissionResource} />
    },
    {
        name: UserProfileGroupsColumnNames.PERMISSION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ResourceLinkHeadPermissionLevel resource={resource as PermissionResource} />
    },
    {
        name: UserProfileGroupsColumnNames.VISIBLE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ResourceLinkTailIsVisible resource={resource as PermissionResource} />
    },
    {
        name: UserProfileGroupsColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ResourceLinkHeadUuid resource={resource as PermissionResource} />
    },
    {
        name: UserProfileGroupsColumnNames.REMOVE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ResourceLinkDelete resource={resource as PermissionResource} />,
    },
];

const ReadOnlyField = withStyles(styles)(
    (props: ({ label: string, input: { value: string } }) & WithStyles<CssRules>) => (
        <Grid item xs={12} data-cy="field">
            <Typography className={props.classes.label}>
                {props.label}
            </Typography>
            <Typography className={props.classes.readOnlyValue} data-cy="value">
                {props.input.value}
            </Typography>
        </Grid>
    )
);

export const UserProfilePanelRoot = withStyles(styles)(
    class extends React.Component<UserProfilePanelRootProps> {
        state = {
            value: TABS.PROFILE,
        };

        componentDidMount() {
            this.setState({ value: TABS.PROFILE });
        }

        render() {
            if (this.props.isInaccessible) {
                return (
                    <Paper className={this.props.classes.emptyRoot}>
                        <CardContent>
                            <DefaultView icon={DetailsIcon} messages={['This user does not exist or your account does not have permission to view it']} />
                        </CardContent>
                    </Paper>
                );
            } else {
                return (
                    <Paper className={this.props.classes.root}>
                        <Tabs value={this.state.value} onChange={this.handleChange} variant={"fullWidth"}>
                            <Tab label={TABS.PROFILE} value={TABS.PROFILE} />
                            <Tab label={TABS.GROUPS} value={TABS.GROUPS} />
                        </Tabs>
                        {this.state.value === TABS.PROFILE &&
                            <CardContent>
                                <Grid container justifyContent="space-between">
                                    <Grid item>
                                        <Typography className={this.props.classes.title}>
                                            {this.props.userUuid}
                                            <CopyToClipboardSnackbar value={this.props.userUuid} />
                                        </Typography>
                                    </Grid>
                                    <Grid item>
                                        <Grid container alignItems="center">
                                            <Grid item style={{ marginRight: '10px' }}><UserResourceAccountStatus uuid={this.props.userUuid} /></Grid>
                                            <Grid item>
                                                <Tooltip title="Actions" disableFocusListener>
                                                    <IconButton
                                                        data-cy='user-profile-panel-options-btn'
                                                        aria-label="Actions"
                                                        onClick={(event) => this.handleContextMenu(event, this.props.user)}
                                                        size="large">
                                                        <MoreVerticalIcon />
                                                    </IconButton>
                                                </Tooltip>
                                            </Grid>
                                        </Grid>
                                    </Grid>
                                </Grid>
                                <form onSubmit={this.props.handleSubmit} data-cy="profile-form">
                                    <Grid container spacing={3}>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12} data-cy="firstName">
                                            <Field
                                                label="First name"
                                                name="firstName"
                                                component={TextField as any}
                                                disabled={!this.props.isAdmin && !this.props.isSelf}
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12} data-cy="lastName">
                                            <Field
                                                label="Last name"
                                                name="lastName"
                                                component={TextField as any}
                                                disabled={!this.props.isAdmin && !this.props.isSelf}
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12} data-cy="email">
                                            <Field
                                                label="E-mail"
                                                name="email"
                                                component={ReadOnlyField as any}
                                                disabled
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12} data-cy="username">
                                            <Field
                                                label="Username"
                                                name="username"
                                                component={ReadOnlyField as any}
                                                disabled
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} xs={12}>
                                            <span className={this.props.classes.userProfileFormMessage}>{this.props.userProfileFormMessage}</span>
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                            <Field
                                                label="Organization"
                                                name="prefs.profile.organization"
                                                component={TextField as any}
                                                disabled={!this.props.isAdmin && !this.props.isSelf}
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                            <Field
                                                label="E-mail at Organization"
                                                name="prefs.profile.organization_email"
                                                component={TextField as any}
                                                disabled={!this.props.isAdmin && !this.props.isSelf}
                                                validate={PROFILE_EMAIL_VALIDATION}
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                            <InputLabel className={this.props.classes.label} htmlFor="prefs.profile.role">Role</InputLabel>
                                            <Field
                                                id="prefs.profile.role"
                                                name="prefs.profile.role"
                                                component={NativeSelectField as any}
                                                items={RoleTypes}
                                                disabled={!this.props.isAdmin && !this.props.isSelf}
                                            />
                                        </Grid>
                                        <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                            <Field
                                                label="Website"
                                                name="prefs.profile.website_url"
                                                component={TextField as any}
                                                disabled={!this.props.isAdmin && !this.props.isSelf}
                                                validate={PROFILE_URL_VALIDATION}
                                            />
                                        </Grid>
                                        <Grid item sm={12}>
                                            <Grid container direction="row" justifyContent="flex-end">
                                                <Button color="primary" onClick={this.props.reset} disabled={this.props.isPristine}>Discard changes</Button>
                                                <Button
                                                    color="primary"
                                                    variant="contained"
                                                    type="submit"
                                                    disabled={this.props.isPristine || this.props.invalid || this.props.submitting}>
                                                    Save changes
                                                </Button>
                                            </Grid>
                                        </Grid>
                                    </Grid>
                                </form >
                            </CardContent>
                        }
                        {this.state.value === TABS.GROUPS &&
                            <div className={this.props.classes.content}>
                                <DataExplorer
                                    id={USER_PROFILE_PANEL_ID}
                                    data-cy="user-profile-groups-data-explorer"
                                    onRowClick={noop}
                                    onRowDoubleClick={noop}
                                    onContextMenu={noop}
                                    contextMenuColumn={false}
                                    hideColumnSelector
                                    hideSearchInput
                                    paperProps={{
                                        elevation: 0,
                                    }}
                                    defaultViewIcon={GroupsIcon}
                                    defaultViewMessages={['Group list is empty.']} />
                            </div>}
                    </Paper >
                );
            }
        }

        handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            this.setState({ value });
        }

        handleContextMenu = (event: React.MouseEvent<HTMLElement>, resource: UserResource | undefined) => {
            event.stopPropagation();
            if (resource) {
                this.props.handleContextMenu(event, resource);
            }
        }

    }
);
