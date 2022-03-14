// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, InjectedFormProps } from "redux-form";
import { TextField } from "components/text-field/text-field";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { NativeSelectField } from "components/select-field/select-field";
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardContent,
    Button,
    Typography,
    Grid,
    InputLabel,
    Tabs, Tab,
    Paper
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { User } from "models/user";
import { DataTableDefaultView } from 'components/data-table-default-view/data-table-default-view';
import { MY_ACCOUNT_VALIDATION } from "validators/validators";
import { USER_PROFILE_PANEL_ID } from 'store/user-profile/user-profile-actions';
import { noop } from 'lodash';
import { GroupsIcon } from 'components/icon/icon';
import { DataColumns } from 'components/data-table/data-table';
import { ResourceLinkHeadUuid, ResourceLinkHeadPermissionLevel, ResourceLinkHead, ResourceLinkDelete, ResourceLinkTailIsVisible } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';

type CssRules = 'root' | 'adminRoot' | 'gridItem' | 'label' | 'readOnlyValue' | 'title' | 'description' | 'actions' | 'content';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    adminRoot: {
        // ...theme.mixins.gutters()
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
        height: `calc(100% - ${theme.spacing.unit * 7}px)`,
    }
});

export interface UserProfilePanelRootActionProps {
    openSetupShellAccount: (uuid: string) => void;
    loginAs: (uuid: string) => void;
    openDeactivateDialog: (uuid: string) => void;
}

export interface UserProfilePanelRootDataProps {
    isAdmin: boolean;
    isSelf: boolean;
    isPristine: boolean;
    isValid: boolean;
    initialValues?: User;
    localCluster: string;
}

const RoleTypes = [
    { key: 'Bio-informatician', value: 'Bio-informatician' },
    { key: 'Data Scientist', value: 'Data Scientist' },
    { key: 'Analyst', value: 'Analyst' },
    { key: 'Researcher', value: 'Researcher' },
    { key: 'Software Developer', value: 'Software Developer' },
    { key: 'System Administrator', value: 'System Administrator' },
    { key: 'Other', value: 'Other' }
];

type UserProfilePanelRootProps = InjectedFormProps<{}> & UserProfilePanelRootActionProps & UserProfilePanelRootDataProps & WithStyles<CssRules>;

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
    ADMIN = "ADMIN",

}

export const userProfileGroupsColumns: DataColumns<string> = [
    {
        name: UserProfileGroupsColumnNames.NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHead uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.PERMISSION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHeadPermissionLevel uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.VISIBLE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTailIsVisible uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHeadUuid uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.REMOVE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkDelete uuid={uuid} />
    },
];

const ReadOnlyField = withStyles(styles)(
    (props: ({ label: string, input: {value: string} }) & WithStyles<CssRules> ) => (
        <Grid item xs={12}>
            <Typography className={props.classes.label}>
                {props.label}
            </Typography>
            <Typography className={props.classes.readOnlyValue}>
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
            this.setState({ value: TABS.PROFILE});
        }

        render() {
            return <Paper className={this.props.classes.root}>
                <Tabs value={this.state.value} onChange={this.handleChange} variant={"fullWidth"}>
                    <Tab label={TABS.PROFILE} value={TABS.PROFILE} />
                    <Tab label={TABS.GROUPS} value={TABS.GROUPS} />
                    {this.props.isAdmin && <Tab label={TABS.ADMIN} value={TABS.ADMIN} />}
                </Tabs>
                {this.state.value === TABS.PROFILE &&
                    <CardContent>
                        <form onSubmit={this.props.handleSubmit}>
                            <Grid container spacing={24}>
                                <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                    <Field
                                        label="First name"
                                        name="firstName"
                                        component={ReadOnlyField as any}
                                        disabled
                                    />
                                </Grid>
                                <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                    <Field
                                        label="Last name"
                                        name="lastName"
                                        component={ReadOnlyField as any}
                                        disabled
                                    />
                                </Grid>
                                <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                    <Field
                                        label="E-mail"
                                        name="email"
                                        component={ReadOnlyField as any}
                                        disabled
                                    />
                                </Grid>
                                <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                    <Field
                                        label="Username"
                                        name="username"
                                        component={ReadOnlyField as any}
                                        disabled
                                    />
                                </Grid>
                                <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                    <Field
                                        label="Organization"
                                        name="prefs.profile.organization"
                                        component={TextField as any}
                                        validate={MY_ACCOUNT_VALIDATION}
                                        disabled={!this.props.isAdmin && !this.props.isSelf}
                                    />
                                </Grid>
                                <Grid item className={this.props.classes.gridItem} sm={6} xs={12}>
                                    <Field
                                        label="E-mail at Organization"
                                        name="prefs.profile.organization_email"
                                        component={TextField as any}
                                        validate={MY_ACCOUNT_VALIDATION}
                                        disabled={!this.props.isAdmin && !this.props.isSelf}
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
                                    />
                                </Grid>
                                <Grid item sm={12}>
                                    <Grid container direction="row" justify="flex-end">
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
                                onRowClick={noop}
                                onRowDoubleClick={noop}
                                contextMenuColumn={false}
                                hideColumnSelector
                                hideSearchInput
                                paperProps={{
                                    elevation: 0,
                                }}
                                dataTableDefaultView={
                                    <DataTableDefaultView
                                        icon={GroupsIcon}
                                        messages={['Group list is empty.']} />
                                } />
                    </div>}
                {this.props.isAdmin && this.state.value === TABS.ADMIN &&
                    <Paper elevation={0} className={this.props.classes.adminRoot}>
                        <Card elevation={0}>
                            <CardContent>
                                <Grid container
                                    direction="row"
                                    justify={'flex-end'}
                                    alignItems={'center'}>
                                    <Grid item xs>
                                        <Typography variant="h6" className={this.props.classes.title}>
                                            Setup Account
                                        </Typography>
                                        <Typography variant="body1" className={this.props.classes.description}>
                                            This button sets up a user. After setup, they will be able use Arvados. This dialog box also allows you to optionally set up a shell account for this user. The login name is automatically generated from the user's e-mail address.
                                        </Typography>
                                    </Grid>
                                    <Grid item sm={'auto'} xs={12}>
                                        <Button variant="contained"
                                            color="primary"
                                            onClick={() => {this.props.openSetupShellAccount(this.props.initialValues.uuid)}}
                                            disabled={false}>
                                            Setup Account
                                        </Button>
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>
                        <Card elevation={0}>
                            <CardContent>
                                <Grid container
                                    direction="row"
                                    justify={'flex-end'}
                                    alignItems={'center'}>
                                    <Grid item xs>
                                        <Typography variant="h6" className={this.props.classes.title}>
                                            Deactivate
                                        </Typography>
                                        <Typography variant="body1" className={this.props.classes.description}>
                                            As an admin, you can deactivate and reset this user. This will remove all repository/VM permissions for the user. If you "setup" the user again, the user will have to sign the user agreement again. You may also want to reassign data ownership.
                                        </Typography>
                                    </Grid>
                                    <Grid item sm={'auto'} xs={12}>
                                        <Button variant="contained"
                                            color="primary"
                                            onClick={() => {this.props.openDeactivateDialog(this.props.initialValues.uuid)}}
                                            disabled={false}>
                                            Deactivate
                                        </Button>
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>
                        <Card elevation={0}>
                            <CardContent>
                                <Grid container
                                    direction="row"
                                    justify={'flex-end'}
                                    alignItems={'center'}>
                                    <Grid item xs>
                                        <Typography variant="h6" className={this.props.classes.title}>
                                            Log In
                                        </Typography>
                                        <Typography variant="body1" className={this.props.classes.description}>
                                            As an admin, you can log in as this user. When you’ve finished, you will need to log out and log in again with your own account.
                                        </Typography>
                                    </Grid>
                                    <Grid item sm={'auto'} xs={12}>
                                        <Button variant="contained"
                                            color="primary"
                                            onClick={() => {this.props.loginAs(this.props.initialValues.uuid)}}
                                            disabled={false}>
                                            Log In
                                        </Button>
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>
                    </Paper>}
            </Paper >;
        }

        handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            this.setState({ value });
        }

    }
);
