// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { UserResource } from '~/models/user';
import { connect, DispatchProp } from 'react-redux';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from '../../services/api/filter-builder';
import { debounce } from 'debounce';
import { ListItemText, Typography } from '@material-ui/core';
import { noop } from 'lodash/fp';

export interface Person {
    name: string;
    email: string;
    uuid: string;
}
export interface PeopleSelectProps {

    items: Person[];

    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onCreate?: (person: Person) => void;
    onDelete?: (index: number) => void;
    onSelect?: (person: Person) => void;

}

export interface PeopleSelectState {
    value: string;
    suggestions: UserResource[];
}

export const PeopleSelect = connect()(
    class PeopleSelect extends React.Component<PeopleSelectProps & DispatchProp, PeopleSelectState> {

        state: PeopleSelectState = {
            value: '',
            suggestions: []
        };

        render() {
            return (
                <Autocomplete
                    label='Invite people'
                    value={this.state.value}
                    items={this.props.items}
                    suggestions={this.state.suggestions}
                    onChange={this.handleChange}
                    onCreate={this.handleCreate}
                    onSelect={this.handleSelect}
                    onDelete={this.handleDelete}
                    onFocus={this.props.onFocus}
                    onBlur={this.props.onBlur}
                    renderChipValue={this.renderChipValue}
                    renderSuggestion={this.renderSuggestion} />
            );
        }

        renderChipValue({ name, uuid }: Person) {
            return name ? name : uuid;
        }

        renderSuggestion({ firstName, lastName, email }: UserResource) {
            return (
                <ListItemText>
                    <Typography noWrap>{`${firstName} ${lastName} <<${email}>>`}</Typography>
                </ListItemText>
            );
        }

        handleDelete = (_: Person, index: number) => {
            const { onDelete = noop } = this.props;
            onDelete(index);
        }

        handleCreate = () => {
            const { onCreate } = this.props;
            if (onCreate) {
                this.setState({ value: '', suggestions: [] });
                onCreate({
                    email: '',
                    name: '',
                    uuid: this.state.value,
                });
            }
        }

        handleSelect = ({ email, firstName, lastName, uuid }: UserResource) => {
            const { onSelect = noop } = this.props;
            this.setState({ value: '', suggestions: [] });
            onSelect({
                email,
                name: `${firstName} ${lastName}`,
                uuid,
            });
        }

        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            this.setState({ value: event.target.value }, this.getSuggestions);
        }

        getSuggestions = debounce(() => this.props.dispatch<any>(this.requestSuggestions), 500);

        requestSuggestions = async (_: void, __: void, { userService }: ServiceRepository) => {
            const { value } = this.state;
            const filters = new FilterBuilder()
                .addILike('email', value)
                .getFilters();
            const { items } = await userService.list({ filters, limit: 5 });
            this.setState({ suggestions: items });
        }

    });
