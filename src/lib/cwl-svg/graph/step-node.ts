import {StepModel} from "cwlts/models";
import {Edge} from "./edge";
import {GraphNode} from "./graph-node";
import {TemplateParser} from "./template-parser";

export class StepNode {

    private svg: SVGSVGElement;
    private stepEl: SVGElement;
    private model: StepModel;

    constructor(element: SVGElement, stepModel: StepModel) {

        this.stepEl = element;
        this.svg    = element.ownerSVGElement!;
        this.model  = stepModel;

    }

    update() {
        const tpl = GraphNode.makeTemplate(this.model);
        const el  = TemplateParser.parse(tpl)!;

        this.stepEl.innerHTML = el.innerHTML;

        // Reposition all edges
        const incomingEdges = this.svg.querySelectorAll(`.edge[data-destination-node="${this.model.connectionId}"]`);
        const outgoingEdges = this.svg.querySelectorAll(`.edge[data-source-node="${this.model.connectionId}"`);

        for (const edge of [...Array.from(incomingEdges), ...Array.from(outgoingEdges)]) {
            Edge.spawnBetweenConnectionIDs(
                this.svg.querySelector(".workflow") as SVGGElement,
                edge.getAttribute("data-source-connection")!,
                edge.getAttribute("data-destination-connection")!
            );
        }

        console.log("Should redraw input port", incomingEdges);

    }
}
