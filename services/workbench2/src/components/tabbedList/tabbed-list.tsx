// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Tabs, Tab, List, ListItem } from '@material-ui/core';

type SingleTabProps = {
    label: string;
    items: any[];
};

type TabbedListProps = {
    tabbedListContents: SingleTabProps[];
    renderListItem?: (item: any) => React.ReactNode;
};

type TabPanelProps = {
    children: React.ReactNode;
    value: number;
    index: number;
};

const TabPanel = ({ children, value, index }: TabPanelProps) => {
    return <div hidden={value !== index}>{value === index && children}</div>;
};

export const TabbedList = ({ tabbedListContents, renderListItem }: TabbedListProps) => {
    const [tabNr, setTabNr] = React.useState(0);

    const handleChange = (event: React.SyntheticEvent, newValue: number) => {
        event.preventDefault();
        setTabNr(newValue);
    };

    return (
        <div>
            <Tabs
                value={tabNr}
                onChange={handleChange}
            >
                {tabbedListContents.map((tab) => (
                    <Tab label={tab.label} />
                ))}
            </Tabs>
            <TabPanel
                value={tabNr}
                index={tabNr}
            >
                <List>
                    {tabbedListContents[tabNr].items.map((item) => (
                      renderListItem ? renderListItem(item) : <ListItem>{item}</ListItem>
                    ))}
                </List>
            </TabPanel>
        </div>
    );
};
