// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Autocomplete } from '~/components/autocomplete/autocomplete';


export interface Person {
    name: string;
}
export interface PeopleSelectProps {
    suggestedPeople: Person[];
}

export interface PeopleSelectState {
    value: string;
    items: Person[];
    suggestions: string[];
}
export class PeopleSelect extends React.Component<PeopleSelectProps, PeopleSelectState> {

    state = {
        value: '',
        items: [{ name: 'Michal Klobukowski' }],
        suggestions: ['Michal Klobukowski', 'Mateusz Ollik']
    };

    render() {
        return (
            <Autocomplete
                label='Invite people'
                value={this.state.value}
                items={this.state.items}
                suggestions={this.getSuggestions()}
                renderChipValue={item => item.name}
                onChange={this.handleChange} />
        );
    }

    getSuggestions() {
        const { value, suggestions } = this.state;
        return value
            ? suggestions.filter(suggestion => suggestion.includes(value))
            : [];
    }

    handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        this.setState({ value: event.target.value });
    }
}
