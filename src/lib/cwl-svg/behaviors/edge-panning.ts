import {Workflow} from "..";

export class EdgePanner {


    /** ID of the requested animation frame for panning */
    private panAnimationFrame: any;

    private workflow: Workflow;

    private movementSpeed = 10;
    private scrollMargin  = 100;

    /**
     * Current state of collision on both axes, each negative if beyond top/left border,
     * positive if beyond right/bottom, zero if inside the viewport
     */
    private collision = {x: 0, y: 0};

    private viewportClientRect: ClientRect;
    private panningCallback = (sdx: number, sdy: number) => {};

    constructor(workflow: Workflow, config = {
        scrollMargin: 100,
        movementSpeed: 10
    }) {
        const options = Object.assign({
            scrollMargin: 100,
            movementSpeed: 10
        }, config);

        this.workflow      = workflow;
        this.scrollMargin  = options.scrollMargin;
        this.movementSpeed = options.movementSpeed;

        this.viewportClientRect = this.workflow.svgRoot.getBoundingClientRect();
    }

    /**
     * Calculates if dragged node is at or beyond the point beyond which workflow panning should be triggered.
     * If collision state has changed, {@link onBoundaryCollisionChange} will be triggered.
     */
    triggerCollisionDetection(x: number, y: number, callback: (sdx: number, sdy: number) => void) {
        const collision      = {x: 0, y: 0};
        this.panningCallback = callback;

        let {left, right, top, bottom} = this.viewportClientRect;

        left   = left + this.scrollMargin;
        right  = right - this.scrollMargin;
        top    = top + this.scrollMargin;
        bottom = bottom - this.scrollMargin;

        if (x < left) {
            collision.x = x - left;
        } else if (x > right) {
            collision.x = x - right;
        }

        if (y < top) {
            collision.y = y - top;
        } else if (y > bottom) {
            collision.y = y - bottom;
        }

        if (
            Math.sign(collision.x) !== Math.sign(this.collision.x)
            || Math.sign(collision.y) !== Math.sign(this.collision.y)
        ) {
            const previous = this.collision;
            this.collision = collision;
            this.onBoundaryCollisionChange(collision, previous);
        }
    }

    /**
     * Triggered when {@link triggerCollisionDetection} determines that collision properties have changed.
     */
    private onBoundaryCollisionChange(current: { x: number, y: number }, previous: { x: number, y: number }): void {

        this.stop();

        if (current.x === 0 && current.y === 0) {
            return;
        }

        this.start(this.collision);
    }

    private start(direction: { x: number, y: number }) {

        let startTimestamp: number | undefined;

        const scale    = this.workflow.scale;
        const matrix   = this.workflow.workflow.transform.baseVal.getItem(0).matrix;
        const sixtyFPS = 16.6666;

        const onFrame = (timestamp: number) => {

            const frameDeltaTime = timestamp - (startTimestamp || timestamp);
            startTimestamp       = timestamp;

            // We need to stop the animation at some point
            // It should be stopped when there is no animation frame ID anymore,
            // which means that stopScroll() was called
            // However, don't do that if we haven't made the first move yet, which is a situation when ∆t is 0
            if (frameDeltaTime !== 0 && !this.panAnimationFrame) {
                startTimestamp = undefined;
                return;
            }

            const moveX = Math.sign(direction.x) * this.movementSpeed * frameDeltaTime / sixtyFPS;
            const moveY = Math.sign(direction.y) * this.movementSpeed * frameDeltaTime / sixtyFPS;

            matrix.e -= moveX;
            matrix.f -= moveY;

            const frameDiffX = moveX / scale;
            const frameDiffY = moveY / scale;

            this.panningCallback(frameDiffX, frameDiffY);
            this.panAnimationFrame = window.requestAnimationFrame(onFrame);
        };

        this.panAnimationFrame = window.requestAnimationFrame(onFrame);
    }

    stop() {
        window.cancelAnimationFrame(this.panAnimationFrame);
        this.panAnimationFrame = undefined;
    }

}
