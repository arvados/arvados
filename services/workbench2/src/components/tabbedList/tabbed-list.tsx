// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useRef } from 'react';
import { Tabs, Tab, List, ListItemButton } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles, withStyles } from '@mui/styles';
import classNames from 'classnames';
import { ArvadosTheme } from 'common/custom-theme';
import { InlinePulser } from 'components/loading/inline-pulser';

type TabbedListClasses = 'root' | 'tabs' | 'listItem' | 'selected' | 'spinner' | 'notFoundLabel' | 'moreResults';

const tabbedListStyles: CustomStyleRulesCallback<TabbedListClasses> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
    },
    tabs: {
        backgroundColor: theme.palette.background.paper,
        position: 'sticky',
        top: 0,
        zIndex: 1,
        borderBottom: '1px solid lightgrey',
    },
    listItem: {
        height: '2rem',
        cursor: 'pointer',
        '&:hover': {
            backgroundColor: theme.palette.grey[200],
        }
    },
    selected: {
        backgroundColor: `${theme.palette.grey['300']} !important`
    },
    spinner: {
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '4rem',
    },
    notFoundLabel: {
        cursor: 'default',
        padding: theme.spacing(1),
        color: theme.palette.grey[700],
        textAlign: 'center',
    },
    moreResults: {
        padding: 0,
        color: theme.palette.grey[700],
        textAlign: 'center',
        fontStyle: 'italic',
        fontSize: '0.8rem',
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
    includeContentsLength: boolean;
    isWorking?: boolean;
    maxLength?: number;
    handleSelect?: (selection: T) => React.MouseEventHandler<HTMLElement> | undefined;
    renderListItem?: (item: T) => React.ReactNode;
    handleTabChange?: (event: React.SyntheticEvent, newValue: number) => void;
};

export const TabbedList = withStyles(tabbedListStyles)(
    <T,>({
        tabbedListContents,
        selectedIndex = 0,
        selectedTab = 0,
        isWorking,
        maxLength,
        injectedStyles,
        classes,
        handleSelect,
        renderListItem,
        handleTabChange,
        includeContentsLength,
    }: TabbedListProps<T> & WithStyles<TabbedListClasses>) => {
    const tabLabels = Object.keys(tabbedListContents);
    const selectedTabLabel = tabLabels[selectedTab];
    const listContents = tabbedListContents[selectedTabLabel] || [];

    const getTabLabel = (label: string) => {
        if (includeContentsLength) { 
            if (maxLength && tabbedListContents[label].length > maxLength) {
                return `${label} (${maxLength}+)`;
            }
            return `${label} (${tabbedListContents[label].length})`;
        } else {
            return label;
        }
    };

    const TabPanel = ({ children, value, index }: TabPanelProps) => {
        return <div hidden={value !== index}>{value === index && children}</div>;
    };

    return (
        <div className={classNames(classes.root, injectedStyles)}>
            <Tabs
                className={classes.tabs}
                value={selectedTab}
                onChange={handleTabChange}
                variant='fullWidth'
            >
                {tabLabels.map((label) => (
                    <Tab key={label} data-cy={`${label}-tab-label`} label={getTabLabel(label)} />
                ))}
            </Tabs>
            <TabPanel
                value={selectedTab}
                index={selectedTab}
            >
                {isWorking ? <div className={classes.spinner}><InlinePulser /></div> :
                    <List dense>
                    {listContents.length === 0 && <div className={classes.notFoundLabel}>no matching {tabLabels[selectedTab]} found</div>}
                        {listContents.slice(0, maxLength).map((item, i) => (
                        <div key={`${selectedTabLabel}-${i}`}>
                            <ListItemButton
                                className={classNames(classes.listItem, { [classes.selected]: i === selectedIndex })}
                                selected={i === selectedIndex}
                                onClick={handleSelect && handleSelect(item)}
                                >
                                {renderListItem ? renderListItem(item) : JSON.stringify(item)}
                            </ListItemButton>
                        </div>
                        ))}
                        {maxLength && listContents.length > maxLength && <div className={classes.moreResults}>{'keep typing to refine search results'}</div>}
                    </List>
                }
            </TabPanel>
        </div>
    );
});
