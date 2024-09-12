// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useRef } from 'react';
import { Tabs, Tab, List, ListItem, StyleRulesCallback, withStyles } from '@material-ui/core';
import { WithStyles } from '@material-ui/core';
import classNames from 'classnames';
import { ArvadosTheme } from 'common/custom-theme';

type TabbedListClasses = 'root' | 'tabs' | 'list' | 'listItem';

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
    listItem: {
        cursor: 'pointer',
    },
});

type TabPanelProps = {
  children: React.ReactNode;
  value: number;
  index: number;
};

type TabbedListProps<T> = {
    tabbedListContents: Record<string, T[]>;
    injectedStyles?: string;
    selectedIndex?: number;
    selectedTab?: number;
    renderListItem?: (item: T) => React.ReactNode;
    handleTabChange?: (event: React.SyntheticEvent, newValue: number) => void;
};

export const TabbedList = withStyles(tabbedListStyles)(<T, _>({ tabbedListContents, selectedIndex, selectedTab, injectedStyles, classes, renderListItem, handleTabChange }: TabbedListProps<T> & WithStyles<TabbedListClasses>) => {
    const tabNr = selectedTab || 0;
    const listRefs = useRef<HTMLDivElement[]>([]);
    const tabLabels = Object.keys(tabbedListContents);

    useEffect(() => {
        if (selectedIndex !== undefined && listRefs.current[selectedIndex]) {
            listRefs.current[selectedIndex].scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }, [selectedIndex]);

    return (
        <div className={classNames(classes.root, injectedStyles)}>
            <div className={classes.tabs}>
                <Tabs
                    value={tabNr}
                    onChange={handleTabChange}
                    fullWidth
                >
                    {tabLabels.map((label) => (
                        <Tab label={label} />
                    ))}
                </Tabs>
            </div>
            <TabPanel
                value={tabNr}
                index={tabNr}
            >
                <List className={classes.list}>
                    {tabbedListContents[tabLabels[tabNr]].map((item, i) => (
                      <div ref={(el) => { if (!!el) listRefs.current[i] = el}}>
                        <ListItem
                        className={classes.listItem}
                        selected={i === selectedIndex}
                        >
                          {renderListItem ? renderListItem(item) : JSON.stringify(item)}
                        </ListItem>
                      </div>
                    ))}
                </List>
            </TabPanel>
        </div>
    );
});

const TabPanel = ({ children, value, index }: TabPanelProps) => {
    return <div hidden={value !== index}>{value === index && children}</div>;
};
