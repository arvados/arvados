// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Tabs, Tab, List, ListItem, StyleRulesCallback, withStyles } from '@material-ui/core';
import { WithStyles } from '@material-ui/core';
import classNames from 'classnames';
import { ArvadosTheme } from 'common/custom-theme';

type TabbedListClasses = 'root' | 'tabs' | 'list';

const tabbedListStyles: StyleRulesCallback<TabbedListClasses> = (theme: ArvadosTheme) => ({
    root: {
      overflowY: 'auto',
    },
    tabs: {
        backgroundColor: theme.palette.background.paper,
        position: 'sticky',
        top: 0,
        zIndex: 1,
        borderBotton: '1px solid lightgrey'
    },
    list: {
        overflowY: 'scroll',
    },
});

type SingleTabProps = {
    label: string;
    items: any[];
};

type TabbedListProps = {
    tabbedListContents: SingleTabProps[];
    renderListItem?: (item: any) => React.ReactNode;
    injectedStyles?: string;
};

type TabPanelProps = {
    children: React.ReactNode;
    value: number;
    index: number;
};

const TabPanel = ({ children, value, index }: TabPanelProps) => {
    return <div hidden={value !== index}>{value === index && children}</div>;
};

export const TabbedList = withStyles(tabbedListStyles)(({ tabbedListContents, renderListItem, injectedStyles, classes }: TabbedListProps & WithStyles<TabbedListClasses>) => {
    const [tabNr, setTabNr] = React.useState(0);

    const handleChange = (event: React.SyntheticEvent, newValue: number) => {
        event.preventDefault();
        setTabNr(newValue);
    };

    return (
        <div className={classNames(classes.root, injectedStyles)}>
            <div className={classes.tabs}>
                <Tabs
                    value={tabNr}
                    onChange={handleChange}
                    fullWidth
                >
                    {tabbedListContents.map((tab) => (
                        <Tab label={tab.label} />
                    ))}
                </Tabs>
            </div>
            <TabPanel
                value={tabNr}
                index={tabNr}
            >
                <List className={classes.list}>{tabbedListContents[tabNr].items.map((item) => (renderListItem ? renderListItem(item) : <ListItem>{item}</ListItem>))}</List>
            </TabPanel>
        </div>
    );
});
