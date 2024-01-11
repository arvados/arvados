// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useRef, useEffect } from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import classnames from 'classnames'
import { ArvadosTheme } from 'common/custom-theme';
import { OverflowMenu } from './ms-toolbar-overflow-menu';

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

export const IntersectionObserverWrapper = withStyles(styles)((props: any & WithStyles<CssRules>) => {
  const { classes, children} = props

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
        }));
    };
    useEffect((): any => {
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
    }, []);

    return (
      <div className={classes.toolbarWrapper} ref={navRef}>
      {React.Children.map(children, (child) => {
        return React.cloneElement(child, {
          className: classnames(child.props.className, {
            [classes.visible]: !!visibilityMap[child.props["data-targetid"]],
            [classes.inVisible]: !visibilityMap[child.props["data-targetid"]]
          })
        });
      })}
      <OverflowMenu
        visibilityMap={visibilityMap}
        className={classes.overflowStyle}
      >
        {children}
      </OverflowMenu>
    </div>
    );
});
