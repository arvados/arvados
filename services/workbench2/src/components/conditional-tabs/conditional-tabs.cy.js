// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ConditionalTabs, TabData } from "./conditional-tabs";
import { Tab } from "@mui/material";

describe("<ConditionalTabs />", () => {
    let tabs = [];
    let WrappedComponent;

    beforeEach(() => {
        tabs = [{
            show: true,
            label: "Tab1",
            content: <div id="content1">Content1</div>,
        },{
            show: false,
            label: "Tab2",
            content: <div id="content2">Content2</div>,
        },{
            show: true,
            label: "Tab3",
            content: <div id="content3">Content3</div>,
        }];

        //necessary to update the props of a component after mounting
        WrappedComponent = ({ tabs }) => {
            const [newTabs, setNewTabs] = React.useState(tabs);

            window.updateProps = (newTabs) => {
                setNewTabs(newTabs);
            };

            return <ConditionalTabs tabs={newTabs} />;
        };
    });

    it("renders only visible tabs", () => {
        // given
        cy.mount(<WrappedComponent tabs={tabs} />);

        // expect 2 visible tabs
        cy.get('button[role=tab]').should('have.length', 2);
        cy.get('button[role=tab]').eq(0).should('contain', 'Tab1');
        cy.get('button[role=tab]').eq(1).should('contain', 'Tab3');

        // expect visible content 1 and tab 3 to be hidden but exist
        // content 2 stays unrendered since the tab is hidden
        cy.contains('Content1').should('exist');
        cy.contains('Content2').should('not.exist');
        cy.contains('Content3').should('have.attr', 'hidden');

        // Show second tab
        cy.window().then((win) => {
            win.updateProps([...tabs, tabs[1].show = true]);
        });

        // Expect 3 visible tabs
        cy.get('button[role=tab]').should('have.length', 3);
        cy.get('button[role=tab]').eq(0).should('contain', 'Tab1');
        cy.get('button[role=tab]').eq(1).should('contain', 'Tab2');
        cy.get('button[role=tab]').eq(2).should('contain', 'Tab3');

        // Expect visible content 1 and hidden content 2/3
        cy.get('div#content1').should('contain', 'Content1');
        cy.get('div#content1').should('not.have.attr', 'hidden');
        cy.get('div#content2').should('have.attr', 'hidden');
        cy.get('div#content3').should('have.attr', 'hidden');

        // Click on Tab2 (position 1)
        cy.get('button[role=tab]').eq(1).click();

        // Expect 3 visible tabs
        cy.get('button[role=tab]').should('have.length', 3);
        cy.get('button[role=tab]').eq(0).should('contain', 'Tab1');
        cy.get('button[role=tab]').eq(1).should('contain', 'Tab2');
        cy.get('button[role=tab]').eq(2).should('contain', 'Tab3');

        // Expect visible content2 and hidden content 1/3
        cy.get('div#content2').should('contain', 'Content2');
        cy.get('div#content1').should('have.attr', 'hidden');
        cy.get('div#content2').should('not.have.attr', 'hidden');
        cy.get('div#content3').should('have.attr', 'hidden');
    });

    it("resets selected tab on tab visibility change", () => {
        // given
        cy.mount(<WrappedComponent tabs={tabs} />);

        // Expect second tab to be Tab3
        cy.get('button[role=tab]').eq(1).should('contain', 'Tab3');
        // Click on Tab3 (position 2)
        cy.get('button[role=tab]').eq(1).click();
        cy.get('div#content3').should('contain', 'Content3');
        cy.get('div#content1').should('have.attr', 'hidden');
        cy.get('div#content2').should('not.exist');
        cy.get('div#content3').should('not.have.attr', 'hidden');

        // Show second tab
        cy.window().then((win) => {
            win.updateProps([...tabs, tabs[1].show = true]);
        });

        // Selected tab resets to 1, tabs 2/3 are hidden
        cy.get('div#content1').should('contain', 'Content1');
        cy.get('div#content1').should('not.have.attr', 'hidden');
        cy.get('div#content2').should('have.attr', 'hidden');
        cy.get('div#content3').should('have.attr', 'hidden');
    });
});
