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
import * as classnames from "classnames";
import Grid from '@material-ui/core/Grid';

function TabContainer(props: any) {
	return (
		<Typography component="div" style={{ padding: 8 * 3 }}>
			{props.children}
		</Typography>
	);
}

export interface DetailsPanelProps {
    toggleDrawer: (isOpened: boolean) => void;
    isOpened: boolean;
}

class DetailsPanel extends React.Component<DetailsPanelProps & WithStyles<CssRules>, {}> {
	state = {
		tabsValue: 0,
	};

	handleChange = (event: any, value: boolean) => {
		this.setState({ tabsValue: value });
	}
	
	render() {
		const { classes, toggleDrawer, isOpened } = this.props;
		const { tabsValue } = this.state;
        return (
            <div className={classes.container}>
				<Drawer variant="persistent" anchor="right" open={isOpened} onClose={() => toggleDrawer(false)}
                    classes={{
                        paper: classes.drawerPaper
                    }}>
					{/* className={classnames([classes.root, { [classes.active]: isActive }])} */}
					<Typography component="div" className={classes.headerContainer}>
						<Grid container alignItems='center' justify='space-around'>
							<Typography variant="title">
								Tutorial pipeline
							</Typography>
							<IconButton color="inherit" onClick={() => toggleDrawer(false)}>
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
					{tabsValue === 0 && <TabContainer>
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
					</TabContainer>}
					{tabsValue === 1 && <TabContainer>
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
					</TabContainer>}
                </Drawer>
            </div>
        );
    }

}

type CssRules = 'drawerPaper' | 'container' | 'headerContainer' 
	| 'tabsIndicator' | 'tabRoot' | 'tabContainer' | 'tabSelected' | 'gridLabel';

const drawerWidth = 320;
const purple = '#692498';
const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
	container: {
		position: 'relative',
		height: 'auto'
	},
    drawerPaper: {
        position: 'relative',
        width: drawerWidth
	},
	headerContainer: {
		color: '#A1A1A1',
		margin: `${theme.spacing.unit}px 0`
	},
	tabsIndicator: {
		backgroundColor: purple
	},
	tabRoot: {
		color: '#333333',
		'&$tabSelected': {
			fontWeight: 700,
			color: purple
		}
	},
	tabContainer: {
		padding: theme.spacing.unit * 3
	},
	tabSelected: {},
	gridLabel: {
		color: '#999999',
	}
});

export default withStyles(styles)(DetailsPanel);