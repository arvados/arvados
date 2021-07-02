import {PluginBase} from "../plugin-base";

export class SVGEdgeHoverPlugin extends PluginBase {

    private boundEdgeEnterFunction = this.onEdgeEnter.bind(this);

    private modelListener: { dispose: Function } = {
        dispose: () => void 0
    };

    afterRender(): void {
        this.attachEdgeHoverBehavior();
    }

    destroy(): void {
        this.detachEdgeHoverBehavior();
        this.modelListener.dispose();
    }

    private attachEdgeHoverBehavior() {

        this.detachEdgeHoverBehavior();
        this.workflow.workflow.addEventListener("mouseenter", this.boundEdgeEnterFunction, true);
    }

    private detachEdgeHoverBehavior() {
        this.workflow.workflow.removeEventListener("mouseenter", this.boundEdgeEnterFunction, true);
    }

    private onEdgeEnter(ev: MouseEvent) {


        // Ignore if we did not enter an edge
        if (!(ev.target! as Element).classList.contains("edge")) return;

        const target = ev.target as SVGGElement;
        let tipEl: SVGGElement;

        const onMouseMove = ((ev: MouseEvent) => {
            const coords = this.workflow.transformScreenCTMtoCanvas(ev.clientX, ev.clientY);
            tipEl.setAttribute("x", String(coords.x));
            tipEl.setAttribute("y", String(coords.y - 16));
        }).bind(this);

        const onMouseLeave = ((ev: MouseEvent) => {
            tipEl.remove();
            target.removeEventListener("mousemove", onMouseMove);
            target.removeEventListener("mouseleave", onMouseLeave)
        }).bind(this);

        this.modelListener = this.workflow.model.on("connection.remove", (source, destination) => {
            if (!tipEl) return;
            const [tipS, tipD] = tipEl.getAttribute("data-source-destination")!.split("$!$");
            if (tipS === source.connectionId && tipD === destination.connectionId) {
                tipEl.remove();
            }
        });

        const sourceNode    = target.getAttribute("data-source-node");
        const destNode      = target.getAttribute("data-destination-node");
        const sourcePort    = target.getAttribute("data-source-port");
        const destPort      = target.getAttribute("data-destination-port");
        const sourceConnect = target.getAttribute("data-source-connection");
        const destConnect   = target.getAttribute("data-destination-connection");

        const sourceLabel = sourceNode === sourcePort ? sourceNode : `${sourceNode} (${sourcePort})`;
        const destLabel   = destNode === destPort ? destNode : `${destNode} (${destPort})`;

        const coords = this.workflow.transformScreenCTMtoCanvas(ev.clientX, ev.clientY);

        const ns = "http://www.w3.org/2000/svg";
        tipEl    = document.createElementNS(ns, "text");
        tipEl.classList.add("label");
        tipEl.classList.add("label-edge");
        tipEl.setAttribute("x", String(coords.x));
        tipEl.setAttribute("y", String(coords.y));
        tipEl.setAttribute("data-source-destination", sourceConnect + "$!$" + destConnect);
        tipEl.innerHTML = sourceLabel + " → " + destLabel;

        this.workflow.workflow.appendChild(tipEl);

        target.addEventListener("mousemove", onMouseMove);
        target.addEventListener("mouseleave", onMouseLeave);

    }

}
