// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classNames from 'classnames';
import { connect } from 'react-redux';
import { CustomizeTableIcon } from 'components/icon/icon';
import { ListItemIcon, StyleRulesCallback, Theme, WithStyles, withStyles, Tooltip, IconButton, Checkbox } from '@material-ui/core';
import { FileTreeData } from '../file-tree/file-tree-data';
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { RootState } from 'store/store';
import { WebDAV, WebDAVRequestConfig } from 'common/webdav';
import { AuthState } from 'store/auth/auth-reducer';
import { extractFilesData } from 'services/collection-service/collection-service-files-response';
import { DefaultIcon, DirectoryIcon, FileIcon } from 'components/icon/icon';
import { setCollectionFiles } from 'store/collection-panel/collection-panel-files/collection-panel-files-actions';

export interface CollectionPanelFilesProps {
    items: any;
    isWritable: boolean;
    isLoading: boolean;
    tooManyFiles: boolean;
    onUploadDataClick: () => void;
    onSearchChange: (searchValue: string) => void;
    onItemMenuOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>, isWritable: boolean) => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, isWritable: boolean) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
    onFileClick: (id: string) => void;
    loadFilesFunc: () => void;
    currentItemUuid: any;
    dispatch: Function;
    collectionPanelFiles: any;
    collectionPanel: any;
}

type CssRules = "wrapper" | "row" | "leftPanel" | "rightPanel" | "pathPanel" | "pathPanelItem" | "rowName" | "listItemIcon" | "rowActive" | "pathPanelMenu" | "rowSelection";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    wrapper: {
        display: 'flex',
    },
    row: {
        display: 'flex',
        margin: '0.5rem',
        cursor: 'pointer',
        "&:hover": {
            backgroundColor: 'rgba(0, 0, 0, 0.08)',
        }
    },
    rowName: {
        paddingTop: '6px',
        paddingBottom: '6px',
    },
    rowSelection: {
        padding: '0px',
    },
    rowActive: {
        color: `${theme.palette.primary.main} !important`,
    },
    listItemIcon: {
        marginTop: '2px',
    },
    pathPanelMenu: {
        float: 'right',
        marginTop: '-15px',
    },
    pathPanel: {
        padding: '1rem',
        marginBottom: '1rem',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    leftPanel: {
        flex: '30%',
        padding: '1rem',
        marginRight: '1rem',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    rightPanel: {
        flex: '70%',
        padding: '1rem',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    pathPanelItem: {
        cursor: 'pointer',
    }

});

export const CollectionPanelFiles = withStyles(styles)(connect((state: RootState) => ({ 
    auth: state.auth,
    collectionPanel: state.collectionPanel,
    collectionPanelFiles: state.collectionPanelFiles,
 }))((props: CollectionPanelFilesProps & WithStyles<CssRules> & { auth: AuthState }) => {
    const { classes, onItemMenuOpen, isWritable, dispatch, collectionPanelFiles, collectionPanel } = props;
    const { apiToken, config } = props.auth;

    const webdavClient = new WebDAV();
    webdavClient.defaults.baseURL = config.keepWebServiceUrl;
    webdavClient.defaults.headers = {
        Authorization: `Bearer ${apiToken}`
    };

    const webDAVRequestConfig: WebDAVRequestConfig = {
        headers: {
            Depth: '1',
        },
    };

    const parentRef = React.useRef(null);
    const [path, setPath]: any = React.useState([]);
    const [pathData, setPathData]: any = React.useState({});
    const [isLoading, setIsLoading] = React.useState(false);

    const leftKey = (path.length > 1 ? path.slice(0, path.length - 1) : path).join('/');
    const rightKey = path.join('/');

    React.useEffect(() => {
        if (props.currentItemUuid) {
            setPathData({});
            setPath([props.currentItemUuid]);
        }
    }, [props.currentItemUuid]);

    React.useEffect(() => {
        if (rightKey && !pathData[rightKey] && !isLoading) {
            webdavClient.propfind(`c=${rightKey}`, webDAVRequestConfig)
                .then((request) => {
                    if (request.responseXML != null) {
                        const result: any = extractFilesData(request.responseXML);
                        const sortedResult = result.sort((n1: any, n2: any) => n1.name > n2.name ? 1 : -1);
                        const newPathData = { ...pathData, [rightKey]: sortedResult };
                        setPathData(newPathData);
                        setIsLoading(false);
                    }
                });
        } else {
            setTimeout(() => setIsLoading(false), 100);
        }
    }, [path, pathData, webdavClient, webDAVRequestConfig, rightKey, isLoading, collectionPanelFiles]);

    const leftData = pathData[leftKey];
    const rightData = pathData[rightKey];

    React.useEffect(() => {
        webdavClient.propfind(`c=${rightKey}`, webDAVRequestConfig)
            .then((request) => {
                if (request.responseXML != null) {
                    const result: any = extractFilesData(request.responseXML);
                    const sortedResult = result.sort((n1: any, n2: any) => n1.name > n2.name ? 1 : -1);
                    const newPathData = { ...pathData, [rightKey]: sortedResult };
                    setPathData(newPathData);
                    setIsLoading(false);
                }
            });
    }, [collectionPanel.item]);

    React.useEffect(() => {
        if (rightData) {
            setCollectionFiles(rightData, false)(dispatch);
        }
    }, [rightData, dispatch]);

    const handleRightClick = React.useCallback(
        (event) => {
            event.preventDefault();

            let elem = event.target;

            while (elem && elem.dataset && !elem.dataset.item) {
                elem = elem.parentNode;
            }

            if (!elem) {
                return;
            }

            const { id } = elem.dataset;
            const item: any = { id, data: rightData.find((elem) => elem.id === id) };

            if (id) {
                onItemMenuOpen(event, item, isWritable);
            }
        },
        [onItemMenuOpen, isWritable, rightData]
    );

    React.useEffect(() => {
        let node = null;

        if (parentRef && parentRef.current) {
            node = parentRef.current;
            (node as any).addEventListener('contextmenu', handleRightClick);
        }

        return () => {
            if (node) {
                (node as any).removeEventListener('contextmenu', handleRightClick);
            }
        };
    }, [parentRef, handleRightClick]);

    const handleClick = React.useCallback(
        (event: any) => {
            let isCheckbox = false;
            let elem = event.target;

            if (elem.type === 'checkbox') {
                isCheckbox = true;
            }

            while (elem && elem.dataset && !elem.dataset.item) {
                elem = elem.parentNode;
            }

            if (elem && elem.dataset && !isCheckbox) {
                const { parentPath, subfolderPath, breadcrumbPath, type } = elem.dataset;

                setIsLoading(true);

                if (breadcrumbPath) {
                    const index = path.indexOf(breadcrumbPath);
                    setPath([...path.slice(0, index + 1)]);
                }

                if (parentPath) {
                    if (path.length > 1) {
                        path.pop()
                    }

                    setPath([...path, parentPath]);
                }

                if (subfolderPath && type === 'directory') {
                    setPath([...path, subfolderPath]);
                }
            }

            if (isCheckbox) {
                const { id } = elem.dataset;
                const item = collectionPanelFiles[id];
                props.onSelectionToggle(event, item);
            }
        },
        [path, setPath, collectionPanelFiles]
    );

    const getItemIcon = React.useCallback(
        (type: string, activeClass: string | null) => {
            let Icon = DefaultIcon;

            switch (type) {
                case 'directory':
                    Icon = DirectoryIcon;
                    break;
                case 'file':
                    Icon = FileIcon;
                    break;
            }

            return (
                <ListItemIcon className={classNames(classes.listItemIcon, activeClass)}>
                    <Icon />
                </ListItemIcon>
            )
        },
        [classes]
    );

    const getActiveClass = React.useCallback(
        (name) => {
            const index = path.indexOf(name);

            return index === (path.length - 1) ? classes.rowActive : null
        },
        [path, classes]
    );

    const onOptionsMenuOpen = React.useCallback(
        (ev, isWritable) => {
            props.onOptionsMenuOpen(ev, isWritable);
        },
        [props.onOptionsMenuOpen]
    );

    return (
        <div onClick={handleClick} ref={parentRef}>
            <div className={classes.pathPanel}>
                {
                    path.map((p: string, index: number) => <span
                        key={`${index}-${p}`}
                        data-item="true"
                        className={classes.pathPanelItem}
                        data-breadcrumb-path={p}
                    >
                        {index === 0 ? 'Home' : p} /&nbsp;
                    </span>)
                }
                <Tooltip  className={classes.pathPanelMenu} title="More options" disableFocusListener>
                    <IconButton
                        data-cy='collection-files-panel-options-btn'
                        onClick={(ev) => onOptionsMenuOpen(ev, isWritable)}>
                        <CustomizeTableIcon />
                    </IconButton>
                </Tooltip>
            </div>
            <div className={classes.wrapper}>
                <div className={classes.leftPanel}>
                    {
                        leftData && !!leftData.length ?
                            leftData.filter(({ type }) => type === 'directory').map(({ name, id, type }: any) => <div
                                data-item="true"
                                data-parent-path={name}
                                className={classNames(classes.row, getActiveClass(name))}
                                key={id}>{getItemIcon(type, getActiveClass(name))} <div className={classes.rowName}>{name}</div>
                            </div>) : <div className={classes.row}>Loading...</div>
                    }
                </div>
                <div className={classes.rightPanel}>
                    {
                        rightData && !isLoading ?
                            rightData.map(({ name, id, type }: any) => <div
                                data-id={id}
                                data-item="true"
                                data-type={type}
                                data-subfolder-path={name}
                                className={classes.row} key={id}>
                                    <Checkbox
                                        color="primary"
                                        className={classes.rowSelection}
                                        checked={collectionPanelFiles[id] ? collectionPanelFiles[id].value.selected : false}
                                    />&nbsp;
                                    {getItemIcon(type, null)} <div className={classes.rowName}>
                                    {name}
                                </div>
                            </div>) : <div className={classes.row}>Loading...</div>
                    }
                </div>
            </div>
        </div>
    );
}));
