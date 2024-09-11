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
        borderBottom: '1px solid lightgrey',
    },
    list: {
        overflowY: 'scroll',
    },
});

type SingleTabProps<T> = {
    label: string;
    items: T[];
};

type TabbedListProps<T> = {
    tabbedListContents: SingleTabProps<T>[];
    renderListItem?: (item: T) => React.ReactNode;
    injectedStyles?: string;
};

type TabPanelProps = {
    children: React.ReactNode;
    value: number;
    index: number;
};

export const TabbedList = withStyles(tabbedListStyles)(<T, _>({ tabbedListContents, renderListItem, injectedStyles, classes }: TabbedListProps<T> & WithStyles<TabbedListClasses>) => {
    const [tabNr, setTabNr] = React.useState(0);

    const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
        event.preventDefault();
        setTabNr(newValue);
    };

    return (
        <div className={classNames(classes.root, injectedStyles)}>
            <div className={classes.tabs}>
                <Tabs
                    value={tabNr}
                    onChange={handleTabChange}
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
                <List className={classes.list}>
                    {tabbedListContents[tabNr].items.map((item) => (
                        <ListItem>{renderListItem ? renderListItem(item) : JSON.stringify(item) }</ListItem>
                    ))}
                </List>
            </TabPanel>
        </div>
    );
});

const TabPanel = ({ children, value, index }: TabPanelProps) => {
    return <div hidden={value !== index}>{value === index && children}</div>;
};
