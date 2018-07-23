// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Drawer, IconButton, Tabs, Tab, Typography, Grid } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import * as classnames from "classnames";
import { connect } from 'react-redux';
import { RootState } from '../../store/store';
import { detailsPanelActions } from "../../store/details-panel/details-panel-action";
import { CloseIcon } from '../../components/icon/icon';
import { EmptyResource } from '../../models/empty';
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { ProjectDetails } from "./project-details";
import { CollectionDetails } from "./collection-details";
import { ProcessDetails } from "./process-details";
import { EmptyDetails } from "./empty-details";
import { DetailsData } from "./details-data";
import { DetailsResource } from "../../models/details";

type CssRules = 'drawerPaper' | 'container' | 'opened' | 'headerContainer' | 'headerIcon' | 'tabContainer';

const drawerWidth = 320;
const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    container: {
        width: 0,
        position: 'relative',
        height: 'auto',
        transition: 'width 0.5s ease',
        '&$opened': {
            width: drawerWidth
        }
    },
    opened: {},
    drawerPaper: {
        position: 'relative',
        width: drawerWidth
    },
    headerContainer: {
        color: theme.palette.grey["600"],
        margin: `${theme.spacing.unit}px 0`,
        textAlign: 'center'
    },
    headerIcon: {
        fontSize: "34px"
    },
    tabContainer: {
        padding: theme.spacing.unit * 3
    }
});

const getItem = (resource: DetailsResource): DetailsData => {
    const res = resource || { kind: undefined, name: 'Projects' };
    switch (res.kind) {
        case ResourceKind.Project:
            return new ProjectDetails(res);
        case ResourceKind.Collection:
            return new CollectionDetails(res);
        case ResourceKind.Process:
            return new ProcessDetails(res);
        default:
            return new EmptyDetails(res as EmptyResource);
    }
};

const mapStateToProps = ({ detailsPanel }: RootState) => ({
    isOpened: detailsPanel.isOpened,
    item: getItem(detailsPanel.item as DetailsResource)
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: () => {
        dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
    }
});

export interface DetailsPanelDataProps {
    onCloseDrawer: () => void;
    isOpened: boolean;
    item: DetailsData;
}

type DetailsPanelProps = DetailsPanelDataProps & WithStyles<CssRules>;

export const DetailsPanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<DetailsPanelProps> {
            state = {
                tabsValue: 0
            };

            handleChange = (event: any, value: boolean) => {
                this.setState({ tabsValue: value });
            }

            renderTabContainer = (children: React.ReactElement<any>) =>
                <Typography className={this.props.classes.tabContainer} component="div">
                    {children}
                </Typography>

            render() {
                const { classes, onCloseDrawer, isOpened, item } = this.props;
                const { tabsValue } = this.state;
                return (
                    <Typography component="div"
                                className={classnames([classes.container, { [classes.opened]: isOpened }])}>
                        <Drawer variant="permanent" anchor="right" classes={{ paper: classes.drawerPaper }}>
                            <Typography component="div" className={classes.headerContainer}>
                                <Grid container alignItems='center' justify='space-around'>
                                    <Grid item xs={2}>
                                        {item.getIcon(classes.headerIcon)}
                                    </Grid>
                                    <Grid item xs={8}>
                                        <Typography variant="title">
                                            {item.getTitle()}
                                        </Typography>
                                    </Grid>
                                    <Grid item>
                                        <IconButton color="inherit" onClick={onCloseDrawer}>
                                            {<CloseIcon/>}
                                        </IconButton>
                                    </Grid>
                                </Grid>
                            </Typography>
                            <Tabs value={tabsValue} onChange={this.handleChange}>
                                <Tab disableRipple label="Details"/>
                                <Tab disableRipple label="Activity" disabled/>
                            </Tabs>
                            {tabsValue === 0 && this.renderTabContainer(
                                <Grid container direction="column">
                                    {item.getDetails()}
                                </Grid>
                            )}
                            {tabsValue === 1 && this.renderTabContainer(
                                <Grid container direction="column"/>
                            )}
                        </Drawer>
                    </Typography>
                );
            }
        }
    )
);
