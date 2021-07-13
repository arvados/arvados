import {Edge as ModelEdge} from "cwlts/models";
import {Geometry} from "../utils/geometry";
import {IOPort} from "./io-port";
import {Workflow} from "./workflow";

export class Edge {

    static makeTemplate(edge: ModelEdge, containerNode: SVGGElement, connectionStates?: string): string | undefined {
        if (!edge.isVisible || edge.source.type === "Step" || edge.destination.type === "Step") {
            return "";
        }

        const [, sourceStepId, sourcePort] = edge.source.id.split("/");
        const [, destStepId, destPort]       = edge.destination.id.split("/");

        const sourceVertex = containerNode.querySelector(`.node[data-id="${sourceStepId}"] .output-port[data-port-id="${sourcePort}"] .io-port`) as SVGGElement;
        const destVertex   = containerNode.querySelector(`.node[data-id="${destStepId}"] .input-port[data-port-id="${destPort}"] .io-port`) as SVGGElement;

        if (edge.source.type === edge.destination.type) {
            console.error("Can't update edge between nodes of the same type.", edge);
            return;
        }

        if (!sourceVertex) {
            console.error("Source vertex not found for edge " + edge.source.id, edge);
            return;
        }

        if (!destVertex) {
            console.error("Destination vertex not found for edge " + edge.destination.id, edge);
            return;
        }

        const sourceCTM = sourceVertex.getCTM() as SVGMatrix;
        const destCTM   = destVertex.getCTM() as SVGMatrix;

        const wfMatrix = containerNode.transform.baseVal.getItem(0).matrix;

        const pathStr = Workflow.makeConnectionPath(
            (sourceCTM.e - wfMatrix.e) / sourceCTM.a,
            (sourceCTM.f - wfMatrix.f) / sourceCTM.a,
            (destCTM.e - wfMatrix.e) / sourceCTM.a,
            (destCTM.f - wfMatrix.f) / sourceCTM.a
        );

        return `
            <g tabindex="-1" class="edge ${connectionStates}"
               data-source-port="${sourcePort}"
               data-destination-port="${destPort}"
               data-source-node="${sourceStepId}"
               data-source-connection="${edge.source.id}"
               data-destination-connection="${edge.destination.id}"
               data-destination-node="${destStepId}">
                <path class="sub-edge outer" d="${pathStr}"></path>
                <path class="sub-edge inner" d="${pathStr}"></path>
            </g>
        `;
    }

    static spawn(pathStr = "", connectionIDs: {
        source?: string,
        destination?: string,
    }                    = {}) {

        const ns   = "http://www.w3.org/2000/svg";
        const edge = document.createElementNS(ns, "g");

        const [, sourceStepId, sourcePort] = (connectionIDs.source || "//").split("/");
        const [, destStepId, destPort]       = (connectionIDs.destination || "//").split("/");

        edge.classList.add("edge");
        if (sourceStepId) {
            edge.classList.add(sourceStepId);
        }
        if (destStepId) {
            edge.classList.add(destStepId);
        }
        edge.setAttribute("tabindex", "-1");
        edge.setAttribute("data-destination-node", destStepId);
        edge.setAttribute("data-destination-port", destPort);
        edge.setAttribute("data-source-port", sourcePort);
        edge.setAttribute("data-source-node", sourceStepId);
        edge.setAttribute("data-source-connection", connectionIDs.source!);
        edge.setAttribute("data-destination-connection", connectionIDs.destination!);

        edge.innerHTML = `
            <path class="sub-edge outer" d="${pathStr}"></path>
            <path class="sub-edge inner" d="${pathStr}"></path>
        `;

        return edge;
    }

    static spawnBetweenConnectionIDs(root: SVGElement, source: string, destination: string) {

        if (source.startsWith("in")) {
            const tmp   = source;
            source      = destination;
            destination = tmp;
        }

        const sourceNode      = root.querySelector(`.port[data-connection-id="${source}"]`) as SVGGElement;
        const destinationNode = root.querySelector(`.port[data-connection-id="${destination}"]`) as SVGAElement;

        const sourceCTM = Geometry.getTransformToElement(sourceNode, root);
        const destCTM   = Geometry.getTransformToElement(destinationNode, root);
        const path      = IOPort.makeConnectionPath(sourceCTM.e, sourceCTM.f, destCTM.e, destCTM.f);

        // If there is already a connection between these ports, update that one instead
        const existingEdge = root.querySelector(`.edge[data-source-connection="${source}"][data-destination-connection="${destination}"]`);
        if (existingEdge) {
            existingEdge.querySelectorAll(".sub-edge").forEach(sub => sub.setAttribute("d", path!));
            return existingEdge;
        }

        const edge = Edge.spawn(path, {
            source,
            destination
        });

        const firstNode = root.querySelector(".node");
        root.insertBefore(edge, firstNode);

        return edge;
    }

    static findEdge(root: any, sourceConnectionID: string, destinationConnectionID: string) {
        return root.querySelector(`[data-source-connection="${sourceConnectionID}"][data-destination-connection="${destinationConnectionID}"]`);
    }

    static parseConnectionID(cid: string) {
        const [side, stepID, portID] = (cid || "//").split("/");
        return {side, stepID, portID};
    }
}
