// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Drawer from '@material-ui/core/Drawer';
import IconButton from "@material-ui/core/IconButton";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import Attribute from '../../components/attribute/attribute';
import Tabs from '@material-ui/core/Tabs';
import Tab from '@material-ui/core/Tab';
import Typography from '@material-ui/core/Typography';
import Grid from '@material-ui/core/Grid';
import * as classnames from "classnames";
import { connect, Dispatch } from 'react-redux';
import EmptyState from '../../components/empty-state/empty-state';
import { RootState } from '../../store/store';
import actions from "../../store/details-panel/details-panel-action";
import { Resource } from '../../common/api/common-resource-service';
import { ResourceKind } from '../../models/kinds';
import { ProjectResource } from '../../models/project';
import { CollectionResource } from '../../models/collection';
import IconBase, { IconTypes } from '../../components/icon/icon';
import { ProcessResource } from '../../models/process';

export interface DetailsPanelDataProps {
    onCloseDrawer: () => void;
    isOpened: boolean;
    icon: IconTypes;
    title: string;
    details: React.ReactElement<any>;
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
        const { classes, onCloseDrawer, isOpened, icon, title, details } = this.props;
        const { tabsValue } = this.state;
        return (
            <Typography component="div" className={classnames([classes.container, { [classes.opened]: isOpened }])}>
                <Drawer variant="permanent" anchor="right" classes={{ paper: classes.drawerPaper }}>
                    <Typography component="div" className={classes.headerContainer}>
                        <Grid container alignItems='center' justify='space-around'>
                            <IconBase className={classes.headerIcon} icon={icon} />
                            <Typography variant="title">
                                {title}
                            </Typography>
                            <IconButton color="inherit" onClick={onCloseDrawer}>
                                <IconBase icon={IconTypes.CLOSE} />
                            </IconButton>
                        </Grid>
                    </Typography>
                    <Tabs value={tabsValue} onChange={this.handleChange}>
                        <Tab disableRipple label="Details" />
                        <Tab disableRipple label="Activity" />
                    </Tabs>
                    {tabsValue === 0 && this.renderTabContainer(
                        <Grid container direction="column">
                            {details}
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
        margin: `${theme.spacing.unit}px 0`
    },
    headerIcon: {
        fontSize: "34px"
    },
    tabContainer: {
        padding: theme.spacing.unit * 3
    }
});

type DetailsPanelResource = ProjectResource | CollectionResource | ProcessResource;

const getIcon = (res: DetailsPanelResource) => {
    switch (res.kind) {
        case ResourceKind.Project:
            return IconTypes.FOLDER;
        case ResourceKind.Collection:
            return IconTypes.COLLECTION;
        case ResourceKind.Process:
            return IconTypes.PROCESS;
        default:
            return IconTypes.FOLDER;
    }
};

const getDetails = (res: DetailsPanelResource) => {
    switch (res.kind) {
        case ResourceKind.Project:
            return <div>
                <Attribute label='Type' value='Project' />
                <Attribute label='Size' value='---' />
                <Attribute label="Location">
                    <IconBase icon={IconTypes.FOLDER} />
                    Projects
                </Attribute>
                <Attribute label='Owner' value='me' />
                <Attribute label='Last modified' value='5:25 PM 5/23/2018' />
                <Attribute label='Created at' value='1:25 PM 5/23/2018' />
                <Attribute label='File size' value='1.4 GB' />
            </div>;
        case ResourceKind.Collection:
            return <div>
                <Attribute label='Type' value='Data Collection' />
                <Attribute label='Size' value='---' />
                <Attribute label="Location">
                    <IconBase icon={IconTypes.FOLDER} />
                    Projects
                </Attribute>
                <Attribute label='Owner' value='me' />
                <Attribute label='Last modified' value='5:25 PM 5/23/2018' />
                <Attribute label='Created at' value='1:25 PM 5/23/2018' />
                <Attribute label='Number of files' value='20' />
                <Attribute label='Content size' value='54 MB' />
                <Attribute label='Collection UUID' link='http://www.google.pl' value='nfnz05wp63ibf8w' />
                <Attribute label='Content address' link='http://www.google.pl' value='nfnz05wp63ibf8w' />
                <Attribute label='Creator' value='Chrystian' />
                <Attribute label='Used by' value='---' />
            </div>;
        case ResourceKind.Process:
            return <div>
                <Attribute label='Type' value='Process' />
                <Attribute label='Size' value='---' />
                <Attribute label="Location">
                    <IconBase icon={IconTypes.FOLDER} />
                    Projects
                </Attribute>
                <Attribute label='Owner' value='me' />
                <Attribute label='Last modified' value='5:25 PM 5/23/2018' />
                <Attribute label='Created at' value='1:25 PM 5/23/2018' />
                <Attribute label='Finished at' value='1:25 PM 5/23/2018' />
                <Attribute label='Outputs' link='http://www.google.pl' value='Container Output' />
                <Attribute label='UUID' link='http://www.google.pl' value='nfnz05wp63ibf8w' />
                <Attribute label='Container UUID' link='http://www.google.pl' value='nfnz05wp63ibf8w' />
                <Attribute label='Priority' value='1' />
                <Attribute label='Runtime constrains' value='1' />
                <Attribute label='Docker image locator' link='http://www.google.pl' value='3838388226321' />
            </div>;
        default:
            return getEmptyState();
    }
};

const getEmptyState = () => {
    return <EmptyState icon={ IconTypes.ANNOUNCEMENT } 
        message='Select a file or folder to view its details.' />;
};

const mapStateToProps = ({ detailsPanel }: RootState) => {
    const { isOpened, item } = detailsPanel;
    return {
        isOpened,
        title: item ? (item as DetailsPanelResource).name : 'Projects',
        icon: item ? getIcon(item as DetailsPanelResource) : IconTypes.FOLDER,
        details: item ? getDetails(item as DetailsPanelResource) : getEmptyState()
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: () => {
        dispatch(actions.TOGGLE_DETAILS_PANEL());
    }
});

const DetailsPanelContainer = connect(mapStateToProps, mapDispatchToProps)(DetailsPanel);

export default withStyles(styles)(DetailsPanelContainer);