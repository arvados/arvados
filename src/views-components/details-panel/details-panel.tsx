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

export interface DetailsPanelDataProps {
    onCloseDrawer: () => void;
    isOpened: boolean;
    header: React.ReactElement<any>;
    renderDetails?: React.ComponentType<{}>;
    renderActivity?: React.ComponentType<{}>;
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
        const { classes, onCloseDrawer, isOpened, header, renderDetails, renderActivity } = this.props;
        const { tabsValue } = this.state;
        return (
            <Typography component="div" className={classnames([classes.container, { [classes.opened]: isOpened }])}>
                <Drawer variant="permanent" anchor="right" classes={{ paper: classes.drawerPaper }}>
                    <Typography component="div" className={classes.headerContainer}>
                        <Grid container alignItems='center' justify='space-around'>
                            {header}
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
                            {renderDetails}
                            <EmptyState icon={IconTypes.ANNOUNCEMENT}
                                message='Select a file or folder to view its details.' />
                            <Attribute label='Type' value='Process' />
                            <Attribute label='Size' value='---' />
                            <Attribute label="Location">
                                <IconBase icon={IconTypes.FOLDER} />
                                Projects
                            </Attribute>
                            <Attribute label='Outputs' link='http://www.google.pl' value='New output as link' />
                            <Attribute label='Owner' value='me' />
                        </Grid>
                    )}
                    {tabsValue === 1 && this.renderTabContainer(
                        <Grid container direction="column">
                            {renderActivity}
                            <EmptyState icon={IconTypes.ANNOUNCEMENT} message='Select a file or folder to view its details.' />
                        </Grid>
                    )}
                </Drawer>
            </Typography>
        );
    }

}

type CssRules = 'drawerPaper' | 'container' | 'opened' | 'headerContainer' | 'tabContainer';

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
        '& .fa-cogs': {
            fontSize: "24px",
            color: theme.customs.colors.green700
        }
    },
    tabContainer: {
        padding: theme.spacing.unit * 3
    }
});

const renderCollectionHeader = (collection: CollectionResource) =>
    <>
        <IconBase icon={IconTypes.COLLECTION} />
        <Typography variant="title">
            {collection.name}
        </Typography>
    </>;

const renderProjectHeader = (project: ProjectResource) =>
    <>
        <IconBase icon={IconTypes.FOLDER} />
        <Typography variant="title">
            {project.name}
        </Typography>
    </>;

const renderHeader = (resource: Resource) => {
    switch(resource.kind) {
        case ResourceKind.Project:
            return renderProjectHeader(resource as ProjectResource);
        case ResourceKind.Collection:
            return renderCollectionHeader(resource as CollectionResource);
        default: 
            return null;
    }
};

const mapStateToProps = ({detailsPanel}: RootState) => ({
    isOpened: detailsPanel.isOpened,
    header: detailsPanel.item ? renderHeader(detailsPanel.item) : null
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: () => {
        dispatch(actions.TOGGLE_DETAILS_PANEL());
    }
});

const DetailsPanelContainer = connect(mapStateToProps, mapDispatchToProps)(DetailsPanel);

export default withStyles(styles)(DetailsPanelContainer);