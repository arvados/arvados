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
		value: 0,
	};

	handleChange = (event: any, value: boolean) => {
		this.setState({ value });
	}
	
	render() {
		const { classes, toggleDrawer, isOpened } = this.props;
		const { value } = this.state;
        return (
            <div className={classes.container}>
				<Drawer variant="persistent" anchor="right" open={isOpened} onClose={() => toggleDrawer(false)}
                    classes={{
                        paper: classes.drawerPaper
                    }}>
					<h2 className={classes.title}>Tutorial pipeline</h2>
					<IconButton color="inherit" onClick={() => toggleDrawer(false)}>
						<CloseIcon />
					</IconButton>
					<Tabs value={value} onChange={this.handleChange}
						classes={{ root: classes.tabsRoot, indicator: classes.tabsIndicator }}>
						<Tab
							disableRipple
							classes={{ root: classes.tabRoot, selected: classes.tabSelected }}
							label="Details" />
						<Tab
							disableRipple
							classes={{ root: classes.tabRoot, selected: classes.tabSelected }}
							label="Activity" />
					</Tabs>
					{value === 0 && <TabContainer>
						Item One
					</TabContainer>}
					{value === 1 && <TabContainer>
						Item Two
					</TabContainer>}
                </Drawer>
            </div>
        );
    }

}

type CssRules = 'drawerPaper' | 'container' | 'title' | 'tabsRoot' | 'tabsIndicator' | 'tabRoot' | 'tabSelected';

const drawerWidth = 320;
const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
	container: {
		position: 'relative',
		height: 'auto'
	},
    drawerPaper: {
        position: 'relative',
        width: drawerWidth
	},
	title: {
		padding: '10px 0px',
		fontSize: '20px',
		fontWeight: 400,
		fontStyle: 'normal'
	},
	tabsRoot: {
		borderBottom: '1px solid transparent',
	},
	tabsIndicator: {
		backgroundColor: 'rgb(106, 27, 154)',
	},
	tabRoot: {
		fontSize: '13px',
		fontWeight: 400,
		color: '#333333',
		'&$tabSelected': {
			fontWeight: 700,
			color: 'rgb(106, 27, 154)'
		},
	},
	tabSelected: {}
});

export default withStyles(styles)(DetailsPanel);