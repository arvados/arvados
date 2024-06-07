// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles, withStyles, ButtonBase, Theme, Popover, Card, Tooltip, IconButton } from "@material-ui/core";
import classnames from "classnames";
import { DefaultTransformOrigin } from "components/popover/helpers";
import { grey } from "@material-ui/core/colors";
import { TCheckedList } from "components/data-table/data-table";

export type CssRules = "root" | "icon" | "iconButton" | "disabled" | "optionsContainer" | "option";

const styles: CustomStyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        borderRadius: "7px",
        "&:hover": {
            backgroundColor: grey[200],
        },
        "&:focus": {
            color: theme.palette.text.primary,
        },
    },
    icon: {
        cursor: "pointer",
        fontSize: 20,
        userSelect: "none",
        "&:hover": {
            color: theme.palette.text.primary,
        },
        paddingBottom: "5px",
    },
    iconButton: {
        color: theme.palette.text.primary,
        opacity: 0.6,
        padding: 1,
        paddingBottom: 5,
    },
    disabled: {
        color: grey[500],
    },
    optionsContainer: {
        padding: "1rem 0",
        flex: 1,
    },
    option: {
        cursor: "pointer",
        display: "flex",
        padding: "3px 2rem",
        fontSize: "0.9rem",
        alignItems: "center",
        "&:hover": {
            backgroundColor: "rgba(0, 0, 0, 0.08)",
        },
    },
});

export type DataTableMultiselectOption = {
    name: string;
    fn: (checkedList) => void;
};

export interface DataTableMultiselectProps {
    name: string;
    disabled: boolean;
    options: DataTableMultiselectOption[];
    checkedList: TCheckedList;
}

interface DataTableFMultiselectPopState {
    anchorEl?: HTMLElement;
}

export const DataTableMultiselectPopover = withStyles(styles)(
    class extends React.Component<DataTableMultiselectProps & WithStyles<CssRules>, DataTableFMultiselectPopState> {
        state: DataTableFMultiselectPopState = {
            anchorEl: undefined,
        };
        icon = React.createRef<HTMLElement>();

        render() {
            const { classes, children, options, checkedList, disabled } = this.props;
            return (
                <>
                    <Tooltip
                        disableFocusListener
                        title="Select options"
                        data-cy="data-table-multiselect-popover"
                    >
                        <ButtonBase
                            className={classnames(classes.root)}
                            component="span"
                            onClick={disabled ? () => {} : this.open}
                            disableRipple
                        >
                            {children}
                            <IconButton
                                component="span"
                                classes={{ root: classes.iconButton }}
                                tabIndex={-1}
                            >
                                <i
                                    className={`${classnames(["fas fa-sort-down", classes.icon])}${disabled ? ` ${classes.disabled}` : ""}`}
                                    data-fa-transform="shrink-3"
                                    ref={this.icon}
                                />
                            </IconButton>
                        </ButtonBase>
                    </Tooltip>
                    <Popover
                        anchorEl={this.state.anchorEl}
                        open={!!this.state.anchorEl}
                        anchorOrigin={DefaultTransformOrigin}
                        transformOrigin={DefaultTransformOrigin}
                        onClose={this.close}
                    >
                        <Card>
                            <div className={classes.optionsContainer}>
                                {options.length &&
                                    options.map((option, i) => (
                                        <div
                                            data-cy={`multiselect-popover-${option.name}`}
                                            key={i}
                                            className={classes.option}
                                            onClick={() => {
                                                option.fn(checkedList);
                                                this.close();
                                            }}
                                        >
                                            {option.name}
                                        </div>
                                    ))}
                            </div>
                        </Card>
                    </Popover>
                </>
            );
        }

        open = () => {
            this.setState({ anchorEl: this.icon.current || undefined });
        };

        close = () => {
            this.setState(prev => ({
                ...prev,
                anchorEl: undefined,
            }));
        };
    }
);
