// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from '~/store/store';
import { getProperty } from '~/store/properties/properties';
import { PROJECT_PANEL_CURRENT_UUID } from '~/store/project-panel/project-panel-action';
import { ArvadosTheme } from '~/common/custom-theme';
import { PopoverOrigin } from '@material-ui/core/Popover';
import { StyleRulesCallback, WithStyles, withStyles, Toolbar, Grid, Button, MenuItem, Menu } from '@material-ui/core';
import { AddIcon, CollectionIcon, ProcessIcon, ProjectIcon } from '~/components/icon/icon';
import { openProjectCreateDialog } from '~/store/projects/project-create-actions';
import { openCollectionCreateDialog } from '~/store/collections/collection-create-actions';
import { matchProjectRoute } from '~/routes/routes';
import { navigateToRunProcess } from '~/store/navigation/navigation-action';
import { runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';

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
    currentItemId: string;
    buttonVisible: boolean;
}

interface SidePanelState {
    anchorEl: any;
}

type SidePanelProps = SidePanelDataProps & DispatchProp & WithStyles<CssRules>;

const transformOrigin: PopoverOrigin = {
    vertical: -50,
    horizontal: 0
};

const isButtonVisible = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProjectRoute(pathname);
    return !!match;
};

export const SidePanelButton = withStyles(styles)(
    connect((state: RootState) => ({
        currentItemId: getProperty(PROJECT_PANEL_CURRENT_UUID)(state.properties),
        buttonVisible: isButtonVisible(state)
    }))(
        class extends React.Component<SidePanelProps> {

            state: SidePanelState = {
                anchorEl: undefined
            };

            render() {
                const { classes, buttonVisible } = this.props;
                const { anchorEl } = this.state;
                return <Toolbar>
                    {buttonVisible && <Grid container>
                        <Grid container item xs alignItems="center" justify="flex-start">
                            <Button variant="contained" color="primary" size="small" className={classes.button}
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
                    </Grid>}
                </Toolbar>;
            }

            handleNewProjectClick = () => {
                this.props.dispatch<any>(openProjectCreateDialog(this.props.currentItemId));
            }

            handleRunProcessClick = () => {
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