// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { push } from 'react-router-redux';
import { LinearProgress, Grid } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { isSystemWorking } from '~/store/progress-indicator/progress-indicator-reducer';
import { RootState } from '~/store/store';
import { User } from '~/models/user';
import { WorkbenchPanel } from '~/views/workbench/workbench';
import { LoginPanel } from '~/views/login-panel/login-panel';
import { MainAppBar } from '~/views-components/main-app-bar/main-app-bar';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        overflow: 'hidden',
        width: '100vw',
        height: '100vh'
    }
});

interface MainPanelDataProps {
    user?: User;
    working: boolean;
}

interface MainPanelGeneralProps {
    buildInfo: string;
}

interface MainPanelState {
    searchText: string;
}

type MainPanelProps = MainPanelDataProps & MainPanelGeneralProps & DispatchProp<any> & WithStyles<CssRules>;

export const MainPanel = withStyles(styles)(
    connect<MainPanelDataProps>(
        (state: RootState) => ({
            user: state.auth.user,
            working: isSystemWorking(state.progressIndicator)
        })
    )(
        class extends React.Component<MainPanelProps, MainPanelState> {
            state = {
                searchText: "",
            };

            render() {
                const { classes, user, buildInfo, working } = this.props;
                const { searchText } = this.state;
                return <>
                    <MainAppBar
                        searchText={searchText}
                        user={user}
                        onSearch={this.onSearch}
                        buildInfo={buildInfo}>
                        {working ? <LinearProgress color="secondary" /> : null}
                    </MainAppBar>
                    <Grid container direction="column" className={classes.root}>
                        {user ? <WorkbenchPanel /> : <LoginPanel />}
                    </Grid>
                </>;
            }

            onSearch = (searchText: string) => {
                this.setState({ searchText });
                this.props.dispatch(push(`/search?q=${searchText}`));
            }
        }
    )
);