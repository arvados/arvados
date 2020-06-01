// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from '~/store/store';
import { ArvadosTheme } from '~/common/custom-theme';
import { PopoverOrigin } from '@material-ui/core/Popover';
import { StyleRulesCallback, WithStyles, withStyles, Toolbar, Grid, Button, MenuItem, Menu } from '@material-ui/core';
import { AddIcon, CollectionIcon, ProcessIcon, ProjectIcon } from '~/components/icon/icon';
import { openProjectCreateDialog } from '~/store/projects/project-create-actions';
import { openCollectionCreateDialog } from '~/store/collections/collection-create-actions';
import { navigateToRunProcess } from '~/store/navigation/navigation-action';
import { runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';
import { getUserUuid } from '~/common/getuser';
import { matchProjectRoute } from '~/routes/routes';
import { GroupResource } from '~/models/group';
import { ResourcesState, getResource } from '~/store/resources/resources';

type CssRules = 'button' | 'menuItem' | 'icon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    button: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
    menuItem: {
        fontSize: '0.875rem',
        color: theme.palette.grey["700"]
    },
    icon: {
        marginRight: theme.spacing.unit
    }
});

interface SidePanelDataProps {
    location: any;
    currentItemId: string;
    resources: ResourcesState;
    currentUserUUID: string | undefined;
}

interface SidePanelState {
    anchorEl: any;
}

type SidePanelProps = SidePanelDataProps & DispatchProp & WithStyles<CssRules>;

const transformOrigin: PopoverOrigin = {
    vertical: -50,
    horizontal: 0
};

export const SidePanelButton = withStyles(styles)(
    connect((state: RootState) => ({
        currentItemId: state.router.location
            ? state.router.location.pathname.split('/').slice(-1)[0]
            : null,
        location: state.router.location,
        resources: state.resources,
        currentUserUUID: getUserUuid(state),
    }))(
        class extends React.Component<SidePanelProps> {

            state: SidePanelState = {
                anchorEl: undefined
            };

            render() {
                const { classes, location, resources, currentUserUUID, currentItemId } = this.props;
                const { anchorEl } = this.state;
                let enabled = false;
                if (currentItemId === currentUserUUID) {
                    enabled = true;
                } else if (matchProjectRoute(location ? location.pathname : '')) {
                    const currentProject = getResource<GroupResource>(currentItemId)(resources);
                    if (currentProject &&
                        currentProject.writableBy.indexOf(currentUserUUID || '') >= 0 &&
                        !currentProject.isTrashed) {
                        enabled = true;
                    }
                }
                return <Toolbar>
                    <Grid container>
                        <Grid container item xs alignItems="center" justify="flex-start">
                            <Button variant="contained" disabled={!enabled}
                                color="primary" size="small" className={classes.button}
                                aria-owns={anchorEl ? 'aside-menu-list' : undefined}
                                aria-haspopup="true"
                                onClick={this.handleOpen}>
                                <AddIcon />
                                New
                            </Button>
                            <Menu
                                id='aside-menu-list'
                                anchorEl={anchorEl}
                                open={Boolean(anchorEl)}
                                onClose={this.handleClose}
                                onClick={this.handleClose}
                                transformOrigin={transformOrigin}>
                                <MenuItem className={classes.menuItem} onClick={this.handleNewCollectionClick}>
                                    <CollectionIcon className={classes.icon} /> New collection
                                </MenuItem>
                                <MenuItem className={classes.menuItem} onClick={this.handleRunProcessClick}>
                                    <ProcessIcon className={classes.icon} /> Run a process
                                </MenuItem>
                                <MenuItem className={classes.menuItem} onClick={this.handleNewProjectClick}>
                                    <ProjectIcon className={classes.icon} /> New project
                                </MenuItem>
                            </Menu>
                        </Grid>
                    </Grid>
                </Toolbar>;
            }

            handleNewProjectClick = () => {
                this.props.dispatch<any>(openProjectCreateDialog(this.props.currentItemId));
            }

            handleRunProcessClick = () => {
                const location = this.props.location;
                this.props.dispatch(runProcessPanelActions.RESET_RUN_PROCESS_PANEL());
                this.props.dispatch(runProcessPanelActions.SET_PROCESS_PATHNAME(location.pathname));
                this.props.dispatch(runProcessPanelActions.SET_PROCESS_OWNER_UUID(this.props.currentItemId));

                this.props.dispatch<any>(navigateToRunProcess);
            }

            handleNewCollectionClick = () => {
                this.props.dispatch<any>(openCollectionCreateDialog(this.props.currentItemId));
            }

            handleClose = () => {
                this.setState({ anchorEl: undefined });
            }

            handleOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
                this.setState({ anchorEl: event.currentTarget });
            }
        }
    )
);