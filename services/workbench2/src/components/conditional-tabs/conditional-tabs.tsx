// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement, useEffect, useState } from "react";
import { Tabs, Tab } from "@material-ui/core";
import { TabsProps } from "@material-ui/core/Tabs";

interface ComponentWithHidden {
    hidden: boolean;
};

export type TabData = {
    show: boolean;
    label: string;
    content: ReactElement<ComponentWithHidden>;
};

type ConditionalTabsProps = {
    tabs: TabData[];
};

export const ConditionalTabs = ({ tabs: tabData, ...props }: Omit<TabsProps, 'value' | 'onChange'> & ConditionalTabsProps) => {
    const [tabState, setTabState] = useState(0);
    const visibleTabs = tabData.filter(tab => tab.show);
    const visibleTabNames = visibleTabs.map(tab => tab.label).join();

    const handleTabChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
        setTabState(value);
    };

    // Reset tab to 0 when tab visibility changes
    // (or if tab set change causes visible set to change)
    useEffect(() => {
        setTabState(0);
    }, [visibleTabNames]);

    return <>
        <Tabs
            {...props}
            value={tabState}
            onChange={handleTabChange} >
            {visibleTabs.map(tab => <Tab key={tab.label} label={tab.label} />)}
        </Tabs>

        {visibleTabs.map((tab, i) => (
            React.cloneElement(tab.content, {key: i, hidden: i !== tabState})
        ))}
    </>;
};
