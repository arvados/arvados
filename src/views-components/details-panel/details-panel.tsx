// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Drawer from '@material-ui/core/Drawer';
import IconButton from "@material-ui/core/IconButton";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import Tabs from '@material-ui/core/Tabs';
import Tab from '@material-ui/core/Tab';
import Typography from '@material-ui/core/Typography';
import Grid from '@material-ui/core/Grid';
import * as classnames from "classnames";
import { connect, Dispatch } from 'react-redux';
import { RootState } from '../../store/store';
import actions from "../../store/details-panel/details-panel-action";
import { ProjectResource } from '../../models/project';
import { CollectionResource } from '../../models/collection';
import IconBase, { IconTypes } from '../../components/icon/icon';
import { ProcessResource } from '../../models/process';
import DetailsPanelFactory from '../../components/details-panel-factory/details-panel-factory';
import AbstractItem from '../../components/details-panel-factory/items/abstract-item';
import { ResourceKind, Resource } from '../../models/resource';
import { EmptyResource } from '../../models/empty';

export interface DetailsPanelDataProps {
    onCloseDrawer: () => void;
    isOpened: boolean;
    item: AbstractItem;
}

type DetailsPanelProps = DetailsPanelDataProps & WithStyles<CssRules>;

class DetailsPanel extends React.Component<DetailsPanelProps, {}> {
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
            <Typography component="div" className={classnames([classes.container, { [classes.opened]: isOpened }])}>
                <Drawer variant="permanent" anchor="right" classes={{ paper: classes.drawerPaper }}>
                    <Typography component="div" className={classes.headerContainer}>
                        <Grid container alignItems='center' justify='space-around'>
                            <Grid item xs={2}>
                                <IconBase className={classes.headerIcon} icon={item.getIcon()} />
                            </Grid>
                            <Grid item xs={8}>
                                <Typography variant="title">
                                    {item.getTitle()}
                                </Typography>
                            </Grid>
                            <Grid item>
                                <IconButton color="inherit" onClick={onCloseDrawer}>
                                    <IconBase icon={IconTypes.CLOSE} />
                                </IconButton>
                            </Grid>
                        </Grid>
                    </Typography>
                    <Tabs value={tabsValue} onChange={this.handleChange}>
                        <Tab disableRipple label="Details" />
                        <Tab disableRipple label="Activity" disabled />
                    </Tabs>
                    {tabsValue === 0 && this.renderTabContainer(
                        <Grid container direction="column">
                            {item.buildDetails()}
                        </Grid>
                    )}
                    {tabsValue === 1 && this.renderTabContainer(
                        <Grid container direction="column" />
                    )}
                </Drawer>
            </Typography>
        );
    }

}

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

// TODO: move to models
export type DetailsPanelResource = ProjectResource | CollectionResource | ProcessResource | EmptyResource;

const getItem = (res: DetailsPanelResource) => {
    return DetailsPanelFactory.createItem(res);
};

const getDefaultItem = () => {
    return DetailsPanelFactory.createItem({ kind: ResourceKind.UNKNOWN, name: 'Projects' } as EmptyResource);
};

const mapStateToProps = ({ detailsPanel }: RootState) => {
    const { isOpened, item } = detailsPanel;
    return {
        isOpened,
        item: item ? getItem(item as DetailsPanelResource) : getDefaultItem()
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: () => {
        dispatch(actions.TOGGLE_DETAILS_PANEL());
    }
});

const DetailsPanelContainer = connect(mapStateToProps, mapDispatchToProps)(DetailsPanel);

export default withStyles(styles)(DetailsPanelContainer);