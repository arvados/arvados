// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Drawer from '@material-ui/core/Drawer';
import IconButton from "@material-ui/core/IconButton";
import CloseIcon from '@material-ui/icons/Close';
import { StyleRulesCallback, WithStyles, withStyles, Theme } from "@material-ui/core/styles";
import Tabs from '@material-ui/core/Tabs';
import Tab from '@material-ui/core/Tab';
import Typography from '@material-ui/core/Typography';
import Grid from '@material-ui/core/Grid';
import * as classnames from "classnames";

export interface DetailsPanelProps {
    closeDrawer: () => void;
    isOpened: boolean;
}

class DetailsPanel extends React.Component<DetailsPanelProps & WithStyles<CssRules>, {}> {
	state = {
		tabsValue: 0,
	};

	handleChange = (event: any, value: boolean) => {
		this.setState({ tabsValue: value });
	}
    
    renderTabContainer = (children: React.ReactElement<any>) => {
        return (
            <Typography className={this.props.classes.tabContainer} component="div">
                {children}
            </Typography>
        );
    }

	render() {
        const { classes, closeDrawer, isOpened } = this.props;
		const { tabsValue } = this.state;
        return (
            <div className={classnames([classes.container, { [classes.opened]: isOpened }])}>
                <Drawer variant="permanent" anchor="right" classes={{ paper: classes.drawerPaper }}>
					<Typography component="div" className={classes.headerContainer}>
						<Grid container alignItems='center' justify='space-around'>
							<Typography variant="title">
								Tutorial pipeline
							</Typography>
                            <IconButton color="inherit" onClick={closeDrawer}>
								<CloseIcon />
							</IconButton>
						</Grid>
					</Typography>
					<Tabs value={tabsValue} onChange={this.handleChange}
						classes={{ indicator: classes.tabsIndicator }}>
						<Tab
							disableRipple
							classes={{ root: classes.tabRoot, selected: classes.tabSelected }}
							label="Details" />
						<Tab
							disableRipple
							classes={{ root: classes.tabRoot, selected: classes.tabSelected }}
							label="Activity" />
					</Tabs>
                    {tabsValue === 0 && this.renderTabContainer(
						<Grid container>
							<Grid item xs={6} sm={4} className={classes.gridLabel}>
								<p>Type</p>
								<p>Size</p>
								<p>Location</p>
								<p>Owner</p>
							</Grid>
							<Grid item xs={6} sm={4}>								
								<p>Process</p>
								<p>---</p>
								<p>Projects</p>
								<p>me</p>
							</Grid>
						</Grid>
					)}
                    {tabsValue === 1 && this.renderTabContainer(
						<Grid container>
							<Grid item xs={6} sm={4} className={classes.gridLabel}>
								<p>Type</p>
								<p>Size</p>
								<p>Location</p>
								<p>Owner</p>
							</Grid>
							<Grid item xs={6} sm={4}>
								<p>Process</p>
								<p>---</p>
								<p>Projects</p>
								<p>me</p>
							</Grid>
						</Grid>
					)}
                </Drawer>
            </div>
        );
    }

}

type CssRules = 'drawerPaper' | 'container' | 'opened' | 'headerContainer' 
	| 'tabsIndicator' | 'tabRoot' | 'tabContainer' | 'tabSelected' | 'gridLabel';

const drawerWidth = 320;
const colorPurple = '#692498';
const colorLightGray = '#A1A1A1';
const colorVeryLightGray = '#999999';
const colorGray = '#333';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
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
        color: colorLightGray,
		margin: `${theme.spacing.unit}px 0`
	},
	tabsIndicator: {
        backgroundColor: colorPurple
	},
	tabRoot: {
        color: colorGray,
		'&$tabSelected': {
			fontWeight: 700,
            color: colorPurple
		}
	},
	tabContainer: {
		padding: theme.spacing.unit * 3
	},
	tabSelected: {},
	gridLabel: {
        color: colorVeryLightGray,
	}
});

export default withStyles(styles)(DetailsPanel);