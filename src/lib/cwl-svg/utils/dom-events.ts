export class DomEvents {

    private handlers = new Map<{ removeEventListener: Function }, { [key: string]: Function[] }>();

    constructor(private root: HTMLElement) {

    }

    public on(event: string, selector: string, handler: (event: UIEvent, target?: Element, root?: Element) => any, root?: Element): Function;
    public on(event: string, handler: (event: UIEvent, target?: Element, root?: Element) => any, root?: Element): Function;
    public on(...args: any[]) {

        const event    = args.shift();
        const selector = typeof args[0] === "string" ? args.shift() : undefined;
        const handler  = typeof args[0] === "function" ? args.shift() : () => {
        };
        const root     = args.shift();

        const eventHolder = root || this.root;

        if (!this.handlers.has(eventHolder)) {
            this.handlers.set(eventHolder, {});
        }
        if (!this.handlers.get(eventHolder)![event]) {
            this.handlers.get(eventHolder)![event] = [];
        }

        const evListener = (ev: UIEvent) => {
            let target: any;
            if (selector) {
                const selected = Array.from(this.root.querySelectorAll(selector));
                target         = ev.target as HTMLElement;
                while (target) {
                    // eslint-disable-next-line
                    if (selected.find(el => el === target)) {
                        break;
                    }
                    target = target.parentNode;
                }

                if (!target) {
                    return;
                }
            }

            const handlerOutput = handler(ev, target || ev.target, this.root);
            if (handlerOutput === false) {
                return false;
            }

            return false;
        };

        eventHolder.addEventListener(event, evListener);

        this.handlers.get(eventHolder)![event].push(evListener);

        return function off() {
            eventHolder.removeEventListener(event, evListener);
        }
    }

    public keyup() {

    }

    public adaptedDrag(selector: string,
                       move?: (dx: number, dy: number, event: UIEvent, target?: Element, root?: Element) => any,
                       start?: (event: UIEvent, target?: Element, root?: Element) => any,
                       end?: (event: UIEvent, target?: Element, root?: Element) => any) {

        let dragging       = false;
        let lastMove: MouseEvent | undefined;
        let draggedEl: Element | undefined;
        let moveEventCount = 0;
        let mouseDownEv: MouseEvent;
        let threshold      = 3;
        let mouseOverListeners: EventListener[];

        const onMouseDown = (ev: MouseEvent, el: Element) => {
            dragging    = true;
            lastMove    = ev;
            draggedEl   = el;
            mouseDownEv = ev;

            ev.preventDefault();

            mouseOverListeners = this.detachHandlers("mouseover");

            document.addEventListener("mousemove", moveHandler);
            document.addEventListener("mouseup", upHandler);

            return false;
        };

        const off = this.on("mousedown", selector, onMouseDown);

        const moveHandler = (ev: MouseEvent) => {
            if (!dragging) {
                return;
            }

            const dx = ev.screenX - lastMove!.screenX;
            const dy = ev.screenY - lastMove!.screenY;
            moveEventCount++;

            if (moveEventCount === threshold && typeof start === "function") {
                start(mouseDownEv, draggedEl, this.root);
            }

            if (moveEventCount >= threshold && typeof move === "function") {
                move(dx, dy, ev, draggedEl, this.root);
            }
        };
        const upHandler   = (ev: MouseEvent) => {
            if (moveEventCount >= threshold) {
                if (dragging) {
                    if (typeof end === "function") {
                        end(ev, draggedEl, this.root)
                    }
                }

                const parentNode        = draggedEl!.parentNode;
                const clickCancellation = (ev: MouseEvent) => {
                    ev.stopPropagation();
                    parentNode!.removeEventListener("click", clickCancellation, true);
                };
                parentNode!.addEventListener("click", clickCancellation, true);
            }

            dragging       = false;
            draggedEl      = undefined;
            lastMove       = undefined;
            moveEventCount = 0;
            document.removeEventListener("mouseup", upHandler);
            document.removeEventListener("mousemove", moveHandler);

            for (let i in mouseOverListeners) {
                this.root.addEventListener("mouseover", mouseOverListeners[i]);
                this.handlers.get(this.root)!["mouseover"] = [];
                this.handlers.get(this.root)!["mouseover"].push(mouseOverListeners[i]);
            }
        };

        return off;
    }


    public drag(selector: string,
                move?: (dx: number, dy: number, event: UIEvent, target?: Element, root?: Element) => any,
                start?: (event: UIEvent, target?: Element, root?: Element) => any,
                end?: (event: UIEvent, target?: Element, root?: Element) => any) {

        let dragging       = false;
        let lastMove: MouseEvent | undefined;
        let draggedEl: Element | undefined;
        let moveEventCount = 0;
        let mouseDownEv: MouseEvent;
        let threshold      = 3;
        let mouseOverListeners: EventListener[];

        const onMouseDown = (ev: MouseEvent, el: Element, root: Element) => {
            dragging    = true;
            lastMove    = ev;
            draggedEl   = el;
            mouseDownEv = ev;

            ev.preventDefault();

            mouseOverListeners = this.detachHandlers("mouseover");

            document.addEventListener("mousemove", moveHandler);
            document.addEventListener("mouseup", upHandler);

            return false;
        };

        const off = this.on("mousedown", selector, onMouseDown);

        const moveHandler = (ev: MouseEvent) => {
            if (!dragging) {
                return;
            }

            const dx = ev.screenX - lastMove!.screenX;
            const dy = ev.screenY - lastMove!.screenY;
            moveEventCount++;

            if (moveEventCount === threshold && typeof start === "function") {
                start(mouseDownEv, draggedEl, this.root);
            }

            if (moveEventCount >= threshold && typeof move === "function") {
                move(dx, dy, ev, draggedEl, this.root);
            }
        };

        const upHandler = (ev: MouseEvent) => {

            if (moveEventCount >= threshold) {
                if (dragging) {
                    if (typeof end === "function") {
                        end(ev, draggedEl, this.root)
                    }
                }

                // When releasing the mouse button, if it happens over the same element that we initially had
                // the mouseDown event, it will trigger a click event. We want to stop that, so we intercept
                // it by capturing click top-down and stopping its propagation.
                // However, if the mouseUp didn't happen above the starting element, it wouldn't trigger a click,
                // but it would intercept the next (unrelated) click event unless we prevent interception in the
                // first place by checking if we released above the starting element.
                if (draggedEl!.contains(ev.target as Node)) {
                    const parentNode = draggedEl!.parentNode;

                    const clickCancellation = (ev: MouseEvent) => {
                        ev.stopPropagation();
                        parentNode!.removeEventListener("click", clickCancellation, true);
                    };
                    parentNode!.addEventListener("click", clickCancellation, true);
                }

            }

            dragging       = false;
            draggedEl      = undefined;
            lastMove       = undefined;
            moveEventCount = 0;
            document.removeEventListener("mouseup", upHandler);
            document.removeEventListener("mousemove", moveHandler);


            for (let i in mouseOverListeners) {
                this.root.addEventListener("mouseover", mouseOverListeners[i]);
                this.handlers.get(this.root)!["mouseover"] = [];
                this.handlers.get(this.root)!["mouseover"].push(mouseOverListeners[i]);
            }
        };

        return off;
    }

    public hover(element: HTMLElement,
                 hover: (event: UIEvent, target?: HTMLElement, root?: HTMLElement) => any = () => {},
                 enter: (event: UIEvent, target?: HTMLElement, root?: HTMLElement) => any = () => {},
                 leave: (event: UIEvent, target?: HTMLElement, root?: HTMLElement) => any = () => {}) {

        let hovering = false;

        element.addEventListener("mouseenter", (ev: MouseEvent) => {
            hovering = true;
            enter(ev, element, this.root);

        });

        element.addEventListener("mouseleave", (ev) => {
            hovering = false;
            leave(ev, element, this.root);
        });

        element.addEventListener("mousemove", (ev) => {
            if (!hovering) {
                return;
            }
            hover(ev, element, this.root);
        });
    }

    public detachHandlers(evName: string, root?: HTMLElement): EventListener[] {
        root                                = root || this.root;
        let eventListeners: EventListener[] = [];
        this.handlers.forEach((handlers: { [event: string]: EventListener[] }, listenerRoot: Element) => {
            if (listenerRoot.id !== root!.id || listenerRoot !== root) {
                return;
            }
            for (let eventName in handlers) {
                if (eventName !== evName) {
                    continue;
                }
                handlers[eventName].forEach((handler) => {
                    eventListeners.push(handler);
                    listenerRoot.removeEventListener(eventName, handler);
                });
            }
        });

        delete this.handlers.get(this.root)![evName];

        return eventListeners;
    }

    public detachAll() {
        this.handlers.forEach((handlers: { [event: string]: EventListener[] }, listenerRoot: Element) => {
            for (let eventName in handlers) {
                handlers[eventName].forEach(handler => listenerRoot.removeEventListener(eventName, handler));
            }
        });

        this.handlers.clear();
    }
}
