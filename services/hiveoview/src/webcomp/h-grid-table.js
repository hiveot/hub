/**
 * GridTable is a light DOM web-component for presenting a sortable table.
 * This is a div element with the 'grid' CSS class.
 *
 * To apply classes to the internal grid container use the 'gridClass' attribute.
 * The default slot is injected into the light DOM so application styles apply.
 *
 * The table layout is managed through a grid CSS which is defined using the
 * title-row fields and their size attributes.
 *
 * @attr striped alternate background each row  (uses css)
 * @attr border include a border around the list
 * @attr title-row for row container with column widths and title formatting
 * @attr gridClass the class to assign to the internal grid container element
 *
 * @class drag-handle to show a drag icon on hover of cells for dragging the row
 *
 * Use draggable="true" on draggable cells in a row or on the row itself.
 * Use 'drag-handle' classname on the element used for dragging
 * For example:
 *  <h-grid-table2 border striped gridClass="">
 *      <li draggable="true" title-row>
 *          <div class=drag-handle>...</div>
 *          <div>stuff</div>
 *      </li>
 *  </h-grid-table2>
 */
const GRID_CLASS_ATTR = "gridClass"
const template = `
<div class="h-grid-table2" >
    <slot>list empty slot content</slot>
</div>

<style>

.dragging {
    opacity: 0.7;
    transform: rotate(-2deg);
}

.h-grid-table2  {
    padding-inline-start:0;
    display: grid;
    /*grid-row-gap:  var(--pico-grid-row-gap);*/
    /*grid-column-gap: var(--pico-grid-column-gap);*/
    /*grid-template-columns is added by addGridColumns */
}

[border] .h-grid-table2 {
    border: 1px solid var(--pico-form-element-border-color);
}

/* Each table row container has a subgrid with the columns from the parent grid
  This avoids the need for display:content which breaks drag&drop.
  Apply row style to all first level children of the grid container.
  */
.h-grid-table2 > * {
  display:grid;
  grid-template-columns:subgrid;
  grid-column: 1/-1; /* magic feature to span all columns */
    background-color: var(--pico-form-element-active-background-color);
}

.h-grid-table2 > * div {
  /*font-weight: var(--pico-font-weight);*/
  /*background-color: var(--pico-card-background-color);*/
  padding: calc(var(--pico-spacing) / 2) calc(var(--pico-spacing) /2);
}

/*Show a grab handle on the cell that is draggable*/
.h-grid-table2 > *  .drag-handle:hover{
    cursor: grab;
}

/* show the insert point when hovering*/
.h-grid-table2 > *.hover {
    border-bottom: 2px solid green !important;
}

[striped] .h-grid-table2 *  {
    background-color: inherit;
}

/* when list is striped*/
[striped] .h-grid-table2 > *:nth-child(even)  {
    background-color: var(--pico-table-row-stripped-background-color);
}

/* title of first row if set*/
/*Can't use the li itself as display is set to contents*/
.h-grid-table2 > *[title-row]   {
    font-variant-caps: small-caps;
    --pico-font-weight: 600;
    border-bottom: 1px solid var(--pico-form-element-active-border-color);
}

</style>
`
// const HList = class extends baseClass {
class HGridTable extends HTMLElement {

    // striped and border work through css
    static get observedAttributes() {
        return [GRID_CLASS_ATTR];
    }

    constructor() {
        super();
        this.draggedEl = undefined
        this.dragEnterCounter = 0 // incr on enter, decr on leave
        this.titleRow = undefined
        this.gridContainer = undefined
    }

    // add the grid-template-columns class to the grid container element
    // this scans the element of the title row for their 'width' attribute
    addGridColumns() {
        let newColumns = ""

        if (!this.titleRow) {
            this.titleRow = this.querySelector('[title-row]');
            if (!this.titleRow) {
                console.log("no row with title-row attribute found")
                return
            }
        }
        // iterate title row children for width. Default to max-content.
        for (let titleCol of this.titleRow.children) {
            let colWidth = titleCol.getAttribute("width")
            if (!colWidth) {
                colWidth = "max-content"
            }
            if (newColumns) {
                newColumns += " "
            }
            newColumns += colWidth
        }
        this.gridContainer.style.gridTemplateColumns = newColumns

    }

    attributeChangedCallback(name, oldValue, newValue) {
        if (name === GRID_CLASS_ATTR) {
            // TODO
        }
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

        // the first child is the actual grid container
        this.gridContainer = this.children[0];

        // this generates the display: grid and grid-template-columns
        this.addGridColumns()

        const gridClassAttr = this.getAttribute(GRID_CLASS_ATTR)
        if (gridClassAttr) {
            let tokens = gridClassAttr.split(" ")
            this.gridContainer.classList.add(...tokens)
        }

        // All rows listen for drag events on all 'li' elements but only 'draggable' items can be dragged
        // let rows = this.querySelectorAll('li[draggable="true"],li[title-row]');
        // let rows = this.querySelectorAll('li');
        let rows = this.gridContainer.children

        for (let row of rows) {
            row.classList.add("h-grid-table2-row")
            // only draggable items can be dragged (eg not the title row)
            row.addEventListener('dragstart', (ev) => {
                console.log("dragstart")
                // a cell in this row starts dragging
                if (ev.target.hasAttribute("draggable")) {
                    this.draggedEl = row
                    ev.dataTransfer.dropEffect = "move"
                    ev.dataTransfer.effectAllowed = "move"
                    ev.dataTransfer.setData("text/html", row.id)
                } else {
                    // this is not a drag start point
                    ev.preventDefault()
                    ev.stopPropagation()
                }
            })

            row.addEventListener('dragover', (ev) => {
                // console.log("dragover")
                // ??? why?
                ev.preventDefault();
            })
            row.addEventListener('dragend', (ev) => {
                console.log("dragend")
                if (this.draggedEl) {
                    this.draggedEl.classList.remove('dragging');
                }
                for (let r of rows) {
                    r.classList.remove("hover")
                }
            })
            row.addEventListener('drop', (ev) => {
                console.log("drop")
                ev.stopPropagation()
                ev.preventDefault();
                // move the dragged element after this element
                if (this.draggedEl !== row) {
                    this.draggedEl.remove()
                    row.after(this.draggedEl)
                }
            })
            row.addEventListener('dragenter', (ev) => {
                // if this is a different target, clear the current target
                if (this.enterTarget && this.enterTarget !== ev.currentTarget) {
                    this.dragEnterCount = 0
                    this.enterTarget.classList.remove("hover")
                }

                this.enterTarget = ev.currentTarget
                this.dragEnterCounter++
                console.log("dragenter; target", row.id)
                // ev.stopPropagation()
                ev.preventDefault()
                row.classList.add("hover")
            })
            row.addEventListener('dragleave', (ev) => {
                this.dragEnterCounter--
                if (this.dragEnterCounter === 0) {
                    console.log("dragleave; target", row.id)
                    // ev.stopPropagation()
                    // ev.preventDefault()
                    row.classList.remove("hover")
                    this.enterTarget = undefined
                }
            })
        }
    }


}

customElements.define('h-grid-table2', HGridTable);
