/**
 * List component presenting a list of sortable items.
 *
 * This is a ul element which listens for drag&drop of li elements.
 * Classes can be used to style the list using the ulClass attribute
 * The default slot is injected into the light DOM so application styles apply.
 *
 * @attr striped alternate background each row  (uses css)
 * @attr border include a border around the list
 * @attr titlerow on li element to format in small caps
 * @attr ulClass the class to assign to the internal ul list element
 *
 * Use draggable="true" for draggable row li elements.
 * Use 'drag-handle' classname on the element used for dragging
 * For example:
 *  <h-list border striped ulClass="">
 *      <li draggable="true" titlerow>
 *          <div class=drag-handle>...</div>
 *          <div>stuff</div>
 *      </li>
 *  </h-list>
 */
const UL_CLASS_ATTR = "ulClass"
const template = `
<ul class="h-list" >
    <slot>list empty slot content</slot>
</ul>

<style>

.dragging {
    opacity: 0.7;
    transform: rotate(-2deg);
}

.h-list  {
padding-inline-start:0;
}

[border] .h-list {
    border: 1px solid var(--pico-form-element-border-color);
}

.h-list li {
    display: flex;
    flex-direction: row;
    gap: var(--pico-spacing);
    list-style: none;
}

/* show the insert point when hovering*/
/*FIXME: support for insert above*/
.h-list li.hover {
    /*background: green !important; !* testing dragging*!*/
    /*Inserts below*/
    border-bottom: 2px solid green !important;
}

/*.h-list li div {*/
/*    font-weight: var(--pico-font-weight);*/
/*    background-color: var(--pico-card-background-color);*/
    /*padding: calc(var(--pico-spacing) / 2) var(--pico-spacing);*/
/*}*/

/* when list is striped*/
[striped] .h-list li:nth-child(even)  {
    background-color: var(--pico-form-element-active-background-color);
}

/* title of first row if set*/
.h-list li[titlerow]   {
    font-variant-caps: small-caps;
    --pico-font-weight: 600;
    border-bottom: 1px solid var(--pico-form-element-active-border-color);
}

</style>
`
// let baseClass = document.createElement('ul').constructor
// const HList = class extends baseClass {
class HList extends HTMLElement {

    constructor() {
        super();
        this.draggedEl = undefined
        this.dragEnterCounter = 0 // incr on enter, decr on leave
        this.ulEl = undefined
    }

    // supports slots in light DOM
    // https://frontendmasters.com/blog/light-dom-only/
    childrenToSlots(html) {
        let templateEl = document.createElement("template");
        templateEl.innerHTML = html;

        // move the slots children into the slots (eg, what shadow root does)
        const slots = templateEl.content.querySelectorAll("slot");
        for (const slot of slots) {
            if (slot.name === "") {
                // the default slot in the template is replaced with the children
                const slotChildren = this.childNodes
                slot.replaceWith(...slotChildren);
                // templateEl.classList = slot.classList
            } else { // named slot
                const slotChildren = this.querySelectorAll(`[slot='${slot.name}']`);
                slot.replaceWith(...slotChildren);
            }
        }
        // finally assign the new template to this container element
        this.replaceChildren(templateEl.content);
        // this.replaceWith(templateEl.content)
    }

    connectedCallback() {
        this.childrenToSlots(template)

        // assign the ulClass attribute
        const ulEl = this.querySelectorAll("ul");

        const ulClassAttr = this.getAttribute(UL_CLASS_ATTR)
        if (ulClassAttr) {
            let tokens = ulClassAttr.split(" ")
            ulEl[0].classList.add(...tokens)
        }

        // All li listen for drag events on all 'li' elements but only 'draggable' items can be dragged
        // let rows = this.querySelectorAll('li[draggable="true"],li[titlerow]');
        let rows = this.querySelectorAll('li');

        for (let i = 0; i < rows.length; i++) {
            const item = rows[i]

            // only draggable items can be dragged (eg not the title row)
            if (item.hasAttribute("draggable")) {
                item.addEventListener('dragstart', (ev) => {
                    console.log("dragstart")
                    this.draggedEl = item
                    ev.dataTransfer.dropEffect = "move"
                    ev.dataTransfer.effectAllowed = "move"
                    ev.dataTransfer.setData("text/html", item.id)
                })
            }
            item.addEventListener('dragover', (ev) => {
                // console.log("dragover")
                // ??? why?
                ev.preventDefault();
            })
            item.addEventListener('dragend', (ev) => {
                console.log("dragend")
                if (this.draggedEl) {
                    this.draggedEl.classList.remove('dragging');
                }
                for (let r of rows) {
                    r.classList.remove("hover")
                }
            })
            item.addEventListener('drop', (ev) => {
                console.log("drop")
                ev.stopPropagation()
                ev.preventDefault();
                // move the dragged element after this element
                if (this.draggedEl !== item) {
                    this.draggedEl.remove()
                    item.after(this.draggedEl)
                }
            })
            item.addEventListener('dragenter', (ev) => {
                // if this is a different target, clear the current target
                if (this.enterTarget && this.enterTarget !== ev.currentTarget ) {
                    this.dragEnterCount = 0
                    this.enterTarget.classList.remove("hover")
                }

                this.enterTarget = ev.currentTarget
                this.dragEnterCounter++
                console.log("dragenter; target", item.id)
                // ev.stopPropagation()
                ev.preventDefault()
                item.classList.add("hover")
            })
            item.addEventListener('dragleave', (ev) => {
                this.dragEnterCounter--
                if (this.dragEnterCounter === 0) {
                    console.log("dragleave; target", item.id)
                    // ev.stopPropagation()
                    // ev.preventDefault()
                    item.classList.remove("hover")
                    this.enterTarget = undefined
                }
            })
        }
    }

    attributeChangedCallback(name, oldValue, newValue) {
        if (name === UL_CLASS_ATTR) {
            // TODO
        }
    }

// striped and border work through css
    static get observedAttributes() {
        return [UL_CLASS_ATTR];
    }

}

customElements.define('h-list', HList);
// customElements.define('h-list', HList, {extends: 'ul'});
