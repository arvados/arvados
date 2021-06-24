import {Edge}          from "cwlts/models";
import {PluginBase}    from "../plugin-base";
import { Workflow } from "lib/cwl-svg";

export class SVGValidatePlugin extends PluginBase {

    private modelDisposers: any[] = [];

    /** Map of CSS classes attached by this plugin */
    private css = {
        plugin: "__plugin-validate",
        invalid: "__validate-invalid"
    };

    registerWorkflow(workflow: Workflow): void {
        super.registerWorkflow(workflow);

        // add plugin specific class to the svgRoot for scoping
        this.workflow.svgRoot.classList.add(this.css.plugin);
    }

    afterModelChange(): void {

        this.disposeModelListeners();

        // add listener for all subsequent edge validation
        const update = this.workflow.model.on("connections.updated", this.renderEdgeValidation.bind(this));
        const create = this.workflow.model.on("connection.create", this.renderEdgeValidation.bind(this));

        this.modelDisposers.concat([update.dispose, create.dispose]);
    }

    destroy(): void {
        this.disposeModelListeners();
    }

    afterRender(): void {
        // do initial validation rendering for edges
        this.renderEdgeValidation();
    }

    onEditableStateChange(enabled: boolean): void {

        if (enabled) {
            // only show validation if workflow is editable
            this.renderEdgeValidation();
        } else {
            this.removeClasses(this.workflow.workflow.querySelectorAll(".edge"))
        }
    }

    private disposeModelListeners(): void {
        for (let disposeListener of this.modelDisposers) {
            disposeListener();
        }
        this.modelDisposers = [];
    }

    private removeClasses(edges: NodeListOf<Element>): void {
        // remove validity class on all edges
        for (const e of (edges as any)) {
            e.classList.remove(this.css.invalid);
        }
    }

    private renderEdgeValidation(): void {
        const graphEdges: any = this.workflow.workflow.querySelectorAll(".edge") as NodeListOf<Element>;

        this.removeClasses(graphEdges);

        // iterate through all modal connections
        this.workflow.model.connections.forEach((e: Edge) => {
            // if the connection isn't valid (should be colored on graph)
            if (!e.isValid) {

                // iterate through edges on the svg
                for (const ge of graphEdges) {
                    const sourceNodeID      = ge.getAttribute("data-source-connection");
                    const destinationNodeID = ge.getAttribute("data-destination-connection");

                    // compare invalid edge source/destination with svg edge
                    if (e.source.id === sourceNodeID && e.destination.id === destinationNodeID) {
                        // if its a match, tag it with the appropriate class and break from the loop
                        ge.classList.add(this.css.invalid);
                        break;
                    }
                }
            }
        });
    }
}
