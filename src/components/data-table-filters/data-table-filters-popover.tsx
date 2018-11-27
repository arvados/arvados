// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
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
    Tooltip
} from "@material-ui/core";
import * as classnames from "classnames";
import { DefaultTransformOrigin } from "~/components/popover/helpers";
import { createTree } from '~/models/tree';
import { DataTableFilters, DataTableFiltersTree } from "./data-table-filters-tree";
import { getNodeDescendants } from '~/models/tree';

export type CssRules = "root" | "icon" | "active" | "checkbox";

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
        '& $icon': {
            opacity: 1,
        },
    },
    icon: {
        marginRight: 4,
        marginLeft: 4,
        opacity: 0.7,
        userSelect: "none",
        width: 16
    },
    checkbox: {
        width: 24,
        height: 24
    }
});

export interface DataTableFilterProps {
    name: string;
    filters: DataTableFilters;
    onChange?: (filters: DataTableFilters) => void;
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
            const { name, classes, children } = this.props;
            const isActive = getNodeDescendants('')(this.state.filters).some(f => f.selected);
            return <>
                <Tooltip title='Filters'>
                    <ButtonBase
                        className={classnames([classes.root, { [classes.active]: isActive }])}
                        component="span"
                        onClick={this.open}
                        disableRipple>
                        {children}
                        <i className={classnames(["fas fa-filter", classes.icon])}
                            data-fa-transform="shrink-3"
                            ref={this.icon} />
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
                            onChange={filters => this.setState({ filters })} />
                        <CardActions>
                            <Button
                                color="primary"
                                variant="raised"
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
