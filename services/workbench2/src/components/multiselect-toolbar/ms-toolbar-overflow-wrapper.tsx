// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useRef, useEffect } from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import classnames from 'classnames';
import { ArvadosTheme } from 'common/custom-theme';
import { OverflowMenu, OverflowChild } from './ms-toolbar-overflow-menu';

type CssRules = 'visible' | 'inVisible' | 'toolbarWrapper' | 'overflowStyle';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    visible: {
        order: 0,
        visibility: 'visible',
        opacity: 1,
    },
    inVisible: {
        order: 100,
        visibility: 'hidden',
        pointerEvents: 'none',
    },
    toolbarWrapper: {
        display: 'flex',
        overflow: 'hidden',
        padding: '0 20px',
        width: '100%',
    },
    overflowStyle: {
        order: 99,
        position: 'sticky',
        right: '-2rem',
        backgroundColor: 'white',
    },
});

type WrapperProps = {
    children: OverflowChild[];
    menuLength: number;
};

export const IntersectionObserverWrapper = withStyles(styles)((props: WrapperProps & WithStyles<CssRules>) => {
    const { classes, children, menuLength } = props;
    const lastEntryId = (children[menuLength - 1] as any).props['data-targetid'];
    const navRef = useRef<any>(null);
    const [visibilityMap, setVisibilityMap] = useState({});

    const handleIntersection = (entries) => {
        const updatedEntries = {};
        entries.forEach((entry) => {
            const targetid = entry.target.dataset.targetid;
            if (entry.isIntersecting) {
                updatedEntries[targetid] = true;
            } else {
                updatedEntries[targetid] = false;
            }
        });

        setVisibilityMap((prev) => ({
                ...prev,
                ...updatedEntries,
                [lastEntryId]: Object.keys(updatedEntries)[0] === lastEntryId,
            })
        );
    };

    useEffect((): any => {
        setVisibilityMap({})
        const observer = new IntersectionObserver(handleIntersection, {
            root: navRef.current,
            rootMargin: '0px -20px 0px 0px',
            threshold: 1,
        });
        // We are adding observers to child elements of the container div
        // with ref as navRef. Notice that we are adding observers
        // only if we have the data attribute targetid on the child element
        if (navRef.current)
            Array.from(navRef.current.children).forEach((item: any) => {
                if (item.dataset.targetid) {
                    observer.observe(item);
                }
            });
        return () => {
            observer.disconnect();
        };
        // eslint-disable-next-line 
    }, [menuLength]);

    const numHidden = (visMap: {}) => {
        return Object.values(visMap).filter((x) => x === false).length;
    };

    return (
        <div
            className={classes.toolbarWrapper}
            ref={navRef}
        >
            {React.Children.map(children, (child) => {
                return React.cloneElement(child, {
                    className: classnames(child.props.className, {
                        [classes.visible]: !!visibilityMap[child.props['data-targetid']],
                        [classes.inVisible]: !visibilityMap[child.props['data-targetid']],
                    }),
                });
            })}
            {numHidden(visibilityMap) >= 2 && (
                <OverflowMenu
                    visibilityMap={visibilityMap}
                    className={classes.overflowStyle}
                >
                    {children}
                </OverflowMenu>
            )}
        </div>
    );
});
