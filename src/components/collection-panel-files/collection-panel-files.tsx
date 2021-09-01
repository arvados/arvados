// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classNames from 'classnames';
import { connect } from 'react-redux';
import { FixedSizeList } from "react-window";
import AutoSizer from "react-virtualized-auto-sizer";
import { CustomizeTableIcon } from 'components/icon/icon';
import { SearchInput } from 'components/search-input/search-input';
import { ListItemIcon, StyleRulesCallback, Theme, WithStyles, withStyles, Tooltip, IconButton, Checkbox, CircularProgress } from '@material-ui/core';
import { FileTreeData } from '../file-tree/file-tree-data';
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { RootState } from 'store/store';
import { WebDAV, WebDAVRequestConfig } from 'common/webdav';
import { AuthState } from 'store/auth/auth-reducer';
import { extractFilesData } from 'services/collection-service/collection-service-files-response';
import { DefaultIcon, DirectoryIcon, FileIcon } from 'components/icon/icon';
import { setCollectionFiles } from 'store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { sortBy } from 'lodash';
import { formatFileSize } from 'common/formatters';

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

type CssRules = "loader" | "wrapper" | "dataWrapper" | "row" | "rowEmpty" | "leftPanel" | "rightPanel" | "pathPanel" | "pathPanelItem" | "rowName" | "listItemIcon" | "rowActive" | "pathPanelMenu" | "rowSelection" | "leftPanelHidden" | "leftPanelVisible" | "searchWrapper" | "searchWrapperHidden";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    wrapper: {
        display: 'flex',
        minHeight: '600px',
        marginBottom: '1rem',
        color: 'rgba(0, 0, 0, 0.87)',
        fontSize: '0.875rem',
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontWeight: 400,
        lineHeight: '1.5',
        letterSpacing: '0.01071em'
    },
    dataWrapper: {
        minHeight: '500px'
    },
    row: {
        display: 'flex',
        marginTop: '0.5rem',
        marginBottom: '0.5rem',
        cursor: 'pointer',
        "&:hover": {
            backgroundColor: 'rgba(0, 0, 0, 0.08)',
        }
    },
    rowEmpty: {
        top: '40%',
        width: '100%',
        textAlign: 'center',
        position: 'absolute'
    },
    loader: {
        top: '50%',
        left: '50%',
        marginTop: '-15px',
        marginLeft: '-15px',
        position: 'absolute'
    },
    rowName: {
        display: 'inline-flex',
        flexDirection: 'column',
        justifyContent: 'center'
    },
    searchWrapper: {
        width: '100%',
        marginBottom: '1rem'
    },
    searchWrapperHidden: {
        width: '0px'
    },
    rowSelection: {
        padding: '0px',
    },
    rowActive: {
        color: `${theme.palette.primary.main} !important`,
    },
    listItemIcon: {
        display: 'inline-flex',
        flexDirection: 'column',
        justifyContent: 'center'
    },
    pathPanelMenu: {
        float: 'right',
        marginTop: '-15px',
    },
    pathPanel: {
        padding: '1rem',
        marginBottom: '1rem',
        backgroundColor: '#fff',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    leftPanel: {
        flex: 0,
        padding: '1rem',
        marginRight: '1rem',
        whiteSpace: 'nowrap',
        position: 'relative',
        backgroundColor: '#fff',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    leftPanelVisible: {
        opacity: 1,
        flex: '30%',
        animation: `animateVisible 1000ms ${theme.transitions.easing.easeOut}`
    },
    leftPanelHidden: {
        opacity: 0,
        flex: 'initial',
        padding: '0',
        marginRight: '0',
    },
    "@keyframes animateVisible": {
        "0%": {
            opacity: 0,
            flex: 'initial',
        },
        "100%": {
            opacity: 1,
            flex: '30%',
        }
    },
    rightPanel: {
        flex: '70%',
        padding: '1rem',
        position: 'relative',
        backgroundColor: '#fff',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    pathPanelItem: {
        cursor: 'pointer',
    }
});

const pathPromise = {};

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
    const [rightClickUsed, setRightClickUsed] = React.useState(false);
    const [leftSearch, setLeftSearch] = React.useState('');
    const [rightSearch, setRightSearch] = React.useState('');

    const leftKey = (path.length > 1 ? path.slice(0, path.length - 1) : path).join('/');
    const rightKey = path.join('/');

    const leftData = (pathData[leftKey] || []).filter(({ type }) => type === 'directory');
    const rightData = pathData[rightKey];

    React.useEffect(() => {
        if (props.currentItemUuid) {
            setPathData({});
            setPath([props.currentItemUuid]);
        }
    }, [props.currentItemUuid]);

    const fetchData = (rightKey, ignoreCache = false) => {
        const dataExists = !!pathData[rightKey];
        const runningRequest = pathPromise[rightKey];

        if ((!dataExists || ignoreCache) && !runningRequest) {
            setIsLoading(true);

            webdavClient.propfind(`c=${rightKey}`, webDAVRequestConfig)
                .then((request) => {
                    if (request.responseXML != null) {
                        const result: any = extractFilesData(request.responseXML);
                        const sortedResult = sortBy(result, (n) => n.name).sort((n1, n2) => {
                            if (n1.type === 'directory' && n2.type !== 'directory') {
                                return -1;
                            }
                            if (n1.type !== 'directory' && n2.type === 'directory') {
                                return 1;
                            }
                            return 0;
                        });
                        const newPathData = { ...pathData, [rightKey]: sortedResult };
                        setPathData(newPathData);
                    }
                })
                .finally(() => {
                    setIsLoading(false);
                    delete pathPromise[rightKey];
                });

            pathPromise[rightKey] = true;
        } else {
            setTimeout(() => setIsLoading(false), 0);
        }
    };

    React.useEffect(() => {
        if (rightKey) {
            fetchData(rightKey);
        }
    }, [rightKey]);

    React.useEffect(() => {
        const hash = (collectionPanel.item || {}).portableDataHash;

        if (hash && rightClickUsed) {
            fetchData(rightKey, true);
        }
    }, [(collectionPanel.item || {}).portableDataHash]);

    React.useEffect(() => {
        if (rightData) {
            setCollectionFiles(rightData, false)(dispatch);
        }
    }, [rightData, dispatch]);

    const handleRightClick = React.useCallback(
        (event) => {
            event.preventDefault();

            if (!rightClickUsed) {
                setRightClickUsed(true);
            }

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
            return path[path.length - 1] === name ? classes.rowActive : null;
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
                    path
                        .map((p: string, index: number) => <span
                            key={`${index}-${p}`}
                            data-item="true"
                            className={classes.pathPanelItem}
                            data-breadcrumb-path={p}
                        >
                            {index === 0 ? 'Home' : p} /&nbsp;
                        </span>)
                }
                <Tooltip className={classes.pathPanelMenu} title="More options" disableFocusListener>
                    <IconButton
                        data-cy='collection-files-panel-options-btn'
                        onClick={(ev) => onOptionsMenuOpen(ev, isWritable)}>
                        <CustomizeTableIcon />
                    </IconButton>
                </Tooltip>
            </div>
            <div className={classes.wrapper}>
                <div className={classNames(classes.leftPanel, path.length > 1 ? classes.leftPanelVisible : classes.leftPanelHidden)}>
                    <div className={path.length > 1 ? classes.searchWrapper : classes.searchWrapperHidden}>
                        <SearchInput label="Search" value={leftSearch} onSearch={setLeftSearch} />
                    </div>
                    <div className={classes.dataWrapper}>
                        {
                            leftData ?
                                <AutoSizer defaultWidth={0}>
                                    {({ height, width }) => {
                                        const filtered = leftData.filter(({ name }) => name.indexOf(leftSearch) > -1);

                                        return !!filtered.length ? <FixedSizeList
                                            height={height}
                                            itemCount={filtered.length}
                                            itemSize={35}
                                            width={width}
                                        >
                                            {
                                                ({ index, style }) => {
                                                    const { id, type, name } = filtered[index];

                                                    return <div
                                                        style={style}
                                                        data-item="true"
                                                        data-parent-path={name}
                                                        className={classNames(classes.row, getActiveClass(name))}
                                                        key={id}>{getItemIcon(type, getActiveClass(name))} <div className={classes.rowName}>{name}</div>
                                                    </div>;
                                                }
                                            }
                                        </FixedSizeList> : <div className={classes.rowEmpty}>No directories available</div>
                                    }}
                                </AutoSizer> : <div className={classes.row}><CircularProgress className={classes.loader} size={30} /></div>
                        }

                    </div>
                </div>
                <div className={classes.rightPanel}>
                    <div className={classes.searchWrapper}>
                        <SearchInput label="Search" value={rightSearch} onSearch={setRightSearch} />
                    </div>
                    <div className={classes.dataWrapper}>
                        {
                            rightData && !isLoading ?
                                <AutoSizer defaultHeight={500}>
                                    {({ height, width }) => {
                                        const filtered = rightData.filter(({ name }) => name.indexOf(rightSearch) > -1);

                                        return !!filtered.length ? <FixedSizeList
                                            height={height}
                                            itemCount={filtered.length}
                                            itemSize={35}
                                            width={width}
                                        >
                                            {
                                                ({ index, style }) => {
                                                    const { id, type, name, size } = filtered[index];

                                                    return <div
                                                        style={style}
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
                                                        <span className={classes.rowName} style={{marginLeft: 'auto', marginRight: '1rem'}}>
                                                            {formatFileSize(size)}
                                                        </span>
                                                    </div>
                                                }
                                            }
                                        </FixedSizeList> : <div className={classes.rowEmpty}>No data available</div>
                                    }}
                                </AutoSizer> : <div className={classes.row}><CircularProgress className={classes.loader} size={30} /></div>
                        }
                    </div>
                </div>
            </div>
        </div>
    );
}));
