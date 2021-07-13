// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import {
    WithStyles,
    withStyles,
    ButtonBase,
    StyleRulesCallback,
    Theme,
    Popover,
    Button,
    Card,
    CardActions,
    Typography,
    CardContent,
    Tooltip,
    IconButton
} from "@material-ui/core";
import classnames from "classnames";
import { DefaultTransformOrigin } from "components/popover/helpers";
import { createTree } from 'models/tree';
import { DataTableFilters, DataTableFiltersTree } from "./data-table-filters-tree";
import { getNodeDescendants } from 'models/tree';

export type CssRules = "root" | "icon" | "iconButton" | "active" | "checkbox";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        cursor: "pointer",
        display: "inline-flex",
        justifyContent: "flex-start",
        flexDirection: "inherit",
        alignItems: "center",
        "&:hover": {
            color: theme.palette.text.primary,
        },
        "&:focus": {
            color: theme.palette.text.primary,
        },
    },
    active: {
        color: theme.palette.text.primary,
        '& $iconButton': {
            opacity: 1,
        },
    },
    icon: {
        fontSize: 12,
        userSelect: 'none',
        width: 16,
        height: 15,
        marginTop: 1
    },
    iconButton: {
        color: theme.palette.text.primary,
        opacity: 0.7,
    },
    checkbox: {
        width: 24,
        height: 24
    }
});

enum SelectionMode {
    ALL = 'all',
    NONE = 'none'
}

export interface DataTableFilterProps {
    name: string;
    filters: DataTableFilters;
    onChange?: (filters: DataTableFilters) => void;

    /**
     * When set to true, only one filter can be selected at a time.
     */
    mutuallyExclusive?: boolean;

    /**
     * By default `all` filters selection means that label should be grayed out.
     * Use `none` when label is supposed to be grayed out when no filter is selected.
     */
    defaultSelection?: SelectionMode;
}

interface DataTableFilterState {
    anchorEl?: HTMLElement;
    filters: DataTableFilters;
    prevFilters: DataTableFilters;
}

export const DataTableFiltersPopover = withStyles(styles)(
    class extends React.Component<DataTableFilterProps & WithStyles<CssRules>, DataTableFilterState> {
        state: DataTableFilterState = {
            anchorEl: undefined,
            filters: createTree(),
            prevFilters: createTree(),
        };
        icon = React.createRef<HTMLElement>();

        render() {
            const { name, classes, defaultSelection = SelectionMode.ALL, children } = this.props;
            const isActive = getNodeDescendants('')(this.state.filters)
                .some(f => defaultSelection === SelectionMode.ALL
                    ? !f.selected
                    : f.selected
                );
            return <>
                <Tooltip disableFocusListener title='Filters'>
                    <ButtonBase
                        className={classnames([classes.root, { [classes.active]: isActive }])}
                        component="span"
                        onClick={this.open}
                        disableRipple>
                        {children}
                        <IconButton component='span' classes={{ root: classes.iconButton }} tabIndex={-1}>
                            <i className={classnames(["fas fa-filter", classes.icon])}
                                data-fa-transform="shrink-3"
                                ref={this.icon} />
                        </IconButton>
                    </ButtonBase>
                </Tooltip>
                <Popover
                    anchorEl={this.state.anchorEl}
                    open={!!this.state.anchorEl}
                    anchorOrigin={DefaultTransformOrigin}
                    transformOrigin={DefaultTransformOrigin}
                    onClose={this.cancel}>
                    <Card>
                        <CardContent>
                            <Typography variant="caption">
                                {name}
                            </Typography>
                        </CardContent>
                        <DataTableFiltersTree
                            filters={this.state.filters}
                            mutuallyExclusive={this.props.mutuallyExclusive}
                            onChange={filters => {
                                this.setState({ filters });
                                if (this.props.mutuallyExclusive) {
                                    const { onChange } = this.props;
                                    if (onChange) {
                                        onChange(filters);
                                    }
                                    this.setState({ anchorEl: undefined });
                                }
                            }} />
                        {this.props.mutuallyExclusive ||
                        <CardActions>
                            <Button
                                color="primary"
                                variant='contained'
                                size="small"
                                onClick={this.submit}>
                                Ok
                            </Button>
                            <Button
                                color="primary"
                                variant="outlined"
                                size="small"
                                onClick={this.cancel}>
                                Cancel
                            </Button>
                        </CardActions >
                        }
                    </Card>
                </Popover>
            </>;
        }

        static getDerivedStateFromProps(props: DataTableFilterProps, state: DataTableFilterState): DataTableFilterState {
            return props.filters !== state.prevFilters
                ? { ...state, filters: props.filters, prevFilters: props.filters }
                : state;
        }

        open = () => {
            this.setState({ anchorEl: this.icon.current || undefined });
        }

        submit = () => {
            const { onChange } = this.props;
            if (onChange) {
                onChange(this.state.filters);
            }
            this.setState({ anchorEl: undefined });
        }

        cancel = () => {
            this.setState(prev => ({
                ...prev,
                filters: prev.prevFilters,
                anchorEl: undefined
            }));
        }

        setFilters = (filters: DataTableFilters) => {
            this.setState({ filters });
        }

    }
);
