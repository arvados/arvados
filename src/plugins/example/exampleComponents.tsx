// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ServiceRepository } from "services/services";
import { Dispatch } from "redux";
import { RootState } from 'store/store';
import { initialize } from 'redux-form';
import { dialogActions } from "store/dialog/dialog-actions";
import { reduxForm, InjectedFormProps, Field, reset, startSubmit } from 'redux-form';
import { TextField } from "components/text-field/text-field";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { withDialog } from "store/dialog/with-dialog";
import { compose } from "redux";
import { propertiesActions } from "store/properties/properties-actions";
import { DispatchProp, connect } from 'react-redux';
import { MenuItem } from "@material-ui/core";
import { Card, CardContent, Typography } from "@material-ui/core";

// This is the name of the dialog box.  It in store actions that
// open/close the dialog box.
export const EXAMPLE_DIALOG_FORM_NAME = "exampleFormName";

// This is the name of the property that will be used to store the
// "pressed" count
export const propertyKey = "Example_menu_item_pressed_count";

// The model backing the form.
export interface ExampleFormDialogData {
    pressedCount: number | string;  // Supposed to start as a number but TextField seems to turn this into a string, unfortunately.
}

// The actual component with the editing fields.  Enables editing
// the 'pressedCount' field.
const ExampleEditFields = () => <span>
    <Field
        name='pressedCount'
        component={TextField}
        type="number"
    />
</span>;

// Callback for when the form is submitted.
const submitEditedPressedCount = (data: ExampleFormDialogData) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(EXAMPLE_DIALOG_FORM_NAME));
        dispatch(propertiesActions.SET_PROPERTY({
            key: propertyKey, value: parseInt(data.pressedCount as string, 10)
        }));
        dispatch(dialogActions.CLOSE_DIALOG({ id: EXAMPLE_DIALOG_FORM_NAME }));
        dispatch(reset(EXAMPLE_DIALOG_FORM_NAME));
    };

// Props for the dialog component
type DialogExampleProps = WithDialogProps<{ updating: boolean }> & InjectedFormProps<ExampleFormDialogData>;

// This is the component that renders the dialog.
const DialogExample = (props: DialogExampleProps) =>
    <FormDialog
        dialogTitle="Edit pressed count"
        formFields={ExampleEditFields}
        submitLabel="Update pressed count"
        {...props}
    />;

// This ties it all together, withDialog() determines if the dialog is
// visible based on state, and reduxForm manages the values of the
// dialog's fields.
export const ExampleDialog = compose(
    withDialog(EXAMPLE_DIALOG_FORM_NAME),
    reduxForm<ExampleFormDialogData>({
        form: EXAMPLE_DIALOG_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(submitEditedPressedCount(data));
        }
    })
)(DialogExample);


// Callback, dispatches an action to set the value of property
// "Example_menu_item_pressed_count"
const incrementPressedCount = (dispatch: Dispatch, pressedCount: number) => {
    dispatch(propertiesActions.SET_PROPERTY({ key: propertyKey, value: pressedCount + 1 }));
};

// Callback, dispatches actions required to initialize and open the
// dialog box.
export const openExampleDialog = (pressedCount: number) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(initialize(EXAMPLE_DIALOG_FORM_NAME, { pressedCount }));
        dispatch(dialogActions.OPEN_DIALOG({
            id: EXAMPLE_DIALOG_FORM_NAME, data: {}
        }));
    };

// Props definition used for menu items.
interface ExampleProps {
    pressedCount: number;
    className?: string;
}

// Called to get the props from the redux state for several of the
// following components.
// Gets the value of the property "Example_menu_item_pressed_count"
// from the state and puts it in 'pressedCount'
const exampleMapStateToProps = (state: RootState) => ({ pressedCount: state.properties[propertyKey] || 0 });

// Define component for the menu item that incremens the count each time it is pressed.
export const ExampleMenuComponent = connect(exampleMapStateToProps)(
    ({ pressedCount, dispatch, className }: ExampleProps & DispatchProp<any>) =>
        <MenuItem className={className} onClick={() => incrementPressedCount(dispatch, pressedCount)}>Example menu item</MenuItem >
);

// Define component for the menu item that opens the dialog box that lets you edit the count directly.
export const ExampleDialogMenuComponent = connect(exampleMapStateToProps)(
    ({ pressedCount, dispatch, className }: ExampleProps & DispatchProp<any>) =>
        <MenuItem className={className} onClick={() => dispatch(openExampleDialog(pressedCount))}>Open example dialog</MenuItem >
);

// The central panel.  Displays the "pressed" count.
export const ExamplePluginMainPanel = connect(exampleMapStateToProps)(
    ({ pressedCount }: ExampleProps) =>
        <Card>
            <CardContent>
                <Typography>
                    This is a example main panel plugin.  The example menu item has been pressed {pressedCount} times.
		</Typography>
            </CardContent>
        </Card>);
