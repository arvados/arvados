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

type TabbedListClasses = 'root' | 'tabs' | 'tabPanel' | 'listItem' | 'selected' | 'spinner' | 'notFoundLabel';

const tabbedListStyles: CustomStyleRulesCallback<TabbedListClasses> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        overflowY: 'auto',
    },
    tabs: {
        backgroundColor: theme.palette.background.paper,
        position: 'sticky',
        top: 0,
        zIndex: 1,
        borderBottom: '1px solid lightgrey',
    },
    tabPanel: {
        overflowY: 'scroll',
    },
    listItem: {
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
    handleSelect?: (selection: T) => React.MouseEventHandler<HTMLElement> | undefined;
    renderListItem?: (item: T) => React.ReactNode;
    handleTabChange?: (event: React.SyntheticEvent, newValue: number) => void;
};

export const TabbedList = withStyles(tabbedListStyles)(<T,>({ tabbedListContents, selectedIndex, selectedTab, isWorking, injectedStyles, classes, handleSelect, renderListItem, handleTabChange, includeContentsLength }: TabbedListProps<T> & WithStyles<TabbedListClasses>) => {
    const tabNr = selectedTab || 0;
    const listRefs = useRef<HTMLDivElement[]>([]);
    const tabLabels = Object.keys(tabbedListContents);

    useEffect(() => {
        if (selectedIndex !== undefined && listRefs.current[selectedIndex]) {
            listRefs.current[selectedIndex].scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }, [selectedIndex]);

    const TabPanel = ({ children, value, index }: TabPanelProps) => {
        return <div className={classes.tabPanel} hidden={value !== index}>{value === index && children}</div>;
    };

    return (
        <div className={classNames(classes.root, injectedStyles)}>
            <div className={classes.tabs}>
                <Tabs
                    value={tabNr}
                    onChange={handleTabChange}
                    variant='fullWidth'
                >
                    {tabLabels.map((label) => (
                        <Tab data-cy={`${label}-tab-label`} label={includeContentsLength ? `${label} (${tabbedListContents[label].length})` : label} />
                    ))}
                </Tabs>
            </div>
            <TabPanel
                value={tabNr}
                index={tabNr}
            >
                {isWorking ? <div className={classes.spinner}><InlinePulser /></div> :
                    <List dense>
                    {tabbedListContents[tabLabels[tabNr]].length === 0 && <div className={classes.notFoundLabel}>no matching {tabLabels[tabNr]} found</div>}
                        {tabbedListContents[tabLabels[tabNr]].map((item, i) => (
                        <div ref={(el) => { if (!!el) listRefs.current[i] = el}}>
                            <ListItemButton
                            className={classNames(classes.listItem, { [classes.selected]: i === selectedIndex })}
                            selected={i === selectedIndex}
                            onClick={handleSelect && handleSelect(item)}
                            >
                            {renderListItem ? renderListItem(item) : JSON.stringify(item)}
                            </ListItemButton>
                        </div>
                        ))}
                    </List>
                }
            </TabPanel>
        </div>
    );
});
