import {Workflow}   from "../..";
import {PluginBase} from "../plugin-base";

export class ZoomPlugin extends PluginBase {
    private svg: SVGSVGElement;
    private dispose: Function | undefined;

    registerWorkflow(workflow: Workflow): void {
        super.registerWorkflow(workflow);
        this.svg = workflow.svgRoot;

        this.dispose = this.attachWheelListener();
    }

    attachWheelListener(): () => void {
        const handler = this.onMouseWheel.bind(this);
        this.svg.addEventListener("mousewheel", handler, true);
        return () => this.svg.removeEventListener("mousewheel", handler, true);
    }

    onMouseWheel(event: MouseWheelEvent) {

        const scale       = this.workflow.scale;
        const scaleUpdate = scale - event.deltaY / 500;

        const zoominOut = scaleUpdate < scale;
        const zoomingIn = scaleUpdate > scale;

        if (zoomingIn && this.workflow.maxScale < scaleUpdate) {
            return;
        }

        if (zoominOut && this.workflow.minScale > scaleUpdate) {
            return;
        }

        this.workflow.scaleAtPoint(scaleUpdate, event.clientX, event.clientY);
        event.stopPropagation();
    }

    destroy(): void {
        if (typeof this.dispose === "function") {
            this.dispose();
        }

        this.dispose = undefined;
    }
}
