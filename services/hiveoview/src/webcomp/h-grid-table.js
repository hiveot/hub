/**
 * GridTable (h-grid-table) is a light DOM web-component for presenting a sortable table.
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
 * @attr gridClass the class to assign to the internal grid container element
 *
 * Grid title row class:
 * @class title-row  to define the column headings and grid widths
 *
 * Grid title-row cell classes:
 * @attr width="100px"  column width. Default is "max-content"
 *
 * Grid data row column classes:
 * @class drag-handle to show a drag icon on hover of cells for dragging the row
 *
 * Use draggable="true" on draggable cells in a row or on the row itself.
 * Use 'drag-handle' classname on the element used for dragging
 * For example:
 *  <h-grid-table border striped gridClass="">
 *      <li draggable="true" class="title-row">
 *          <div class=drag-handle>...</div>
 *          <div>stuff</div>
 *      </li>
 *  </h-grid-table>
 */
const GRID_CLASS_ATTR = "gridClass"
const template = `
<ul class="h-grid-table" >
    <slot>list empty slot content</slot>
</ul>

<style id="h-grid-table-style">

.dragging {
    opacity: 0.7;
    transform: rotate(-2deg);
}

.h-grid-table  {
    padding-inline-start:0;
    display: grid;
    /*grid-row-gap:  var(--pico-grid-row-gap);*/
    /*grid-column-gap: var(--pico-grid-column-gap);*/
    /*grid-template-columns is added by addGridColumns */
}

[border] .h-grid-table {
    border: 1px solid var(--pico-form-element-border-color);
}

/* Each table row container has a subgrid with the columns from the parent grid
  This avoids the need for display:content which breaks drag&drop.
  Apply row style to all first level children of the grid container.
  */
.h-grid-table > * {
  display:grid;
  grid-template-columns:subgrid;
  grid-column: 1/-1; /* span all columns */
    background-color: var(--pico-form-element-active-background-color);
}

.h-grid-table > * div {
  /*font-weight: var(--pico-font-weight);*/
  /*background-color: var(--pico-card-background-color);*/
  padding: calc(var(--pico-spacing) / 2) calc(var(--pico-spacing) /2);
}

/*Show a grab handle on the cell that is draggable*/
.h-grid-table > *  .drag-handle:hover{
    cursor: grab;
}

/* show the insert point when hovering*/
.h-grid-table > *.hover {
    border-bottom: 2px solid green !important;
}

[striped] .h-grid-table *  {
    background-color: inherit;
}

/* when list is striped*/
[striped] .h-grid-table > *:nth-child(even)  {
    background-color: var(--pico-table-row-stripped-background-color);
}

/* title of first row if set*/
/*Can't use the li itself as display is set to contents*/
.h-grid-table > *.title-row   {
    font-variant-caps: small-caps;
    --pico-font-weight: 600;
    border-bottom: 1px solid var(--pico-form-element-active-border-color);
}


/*1. Establish class of grid table instance (use id?)*/
/*2. add instance class rule to global stylesheet*/

/* testing: hide 4rd col*/
/*.edit-tile-sources > li > *:nth-child(4) {*/
/*    display: var(--h-show-md);*/
/*}*/


</style>
`

// const HList = class extends baseClass {
//
// This component adds style rules for instances. To avoid a rampart growth of
// duplicate rules, style rules are only added once. This is tracked in the
// class prototype.
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

    // Create a stylesheet containing media queries for the grid-template-column.
    //
    // This uses the title row attributes. 'show' and 'width':
    //   'show=' the media threshold in px or name (sm, md, lg, xl)
    //   'width=' the css grid column size. Default is 'auto'.
    //
    // <h-grid-table id="my-table">
    //    <li>
    //       <div show="640px" width="200px"></div>
    //       <div width="auto"></div>
    //       <div show="lg" width="1fr"></div>
    //    </li>
    //    ...  data rows ...
    // </h-grid-table>
    //
    // The generated stylesheet:
    //
    // First the default with all columns:
    // #grid-table-id {
    //    grid-template-columns:  640px auto 1fr
    // }
    //
    // then media queries for hiding columns:
    //
    // @media (screen < 640px) {
    //    #grid-table-id > li > :nth-child(1) {
    //       display: none
    //    }
    //    grid-template-columns:  auto;
    // }
    // @media (screen < lg) {
    //    #grid-table-id > li > :nth-child(3) {
    //       display: none
    //    }
    //    grid-template-columns:  640 auto;
    // }
    createStyleSheet() {
        let allColumnSizes = ""
        let gridId = this.gridContainer.id
        this.gridContainer.classList.add(gridId)
        let styleSheet = document.getElementById("h-grid-table-style").sheet

        // Avoid duplicate styles for the same gridId using a custom property map.
        // This map is stored in the same sheet as the styles.
        let styleTracker = styleSheet["styleTracker"]
        if (!styleTracker) {
            styleTracker = {}
            styleSheet["styleTracker"] = styleTracker
        }
        if (!styleTracker[gridId]) {
            styleTracker[gridId] = "styles-are-set"
        } else {
            // styles have already been set
            return
        }

        // determine the default grid-template-columns. Additional overrides
        // are added in media-queries. Default width is auto.
        for (let i = 0; i < this.titleRow.children.length; i++) {
            const titleCol = this.titleRow.children[i]
            let width = titleCol.getAttribute("width")
            if (!width) {
                width = "auto"
            }
            allColumnSizes += " " + width
        }
        // determine the media styles
        let mediaStyleSheet = ""
        let mediaStyle = ""
        let xsCols = []
        let smCols = []
        let mdCols = []
        let lgCols = []
        let xlCols = []
        for (let i = 0; i < this.titleRow.children.length; i++) {
            const titleCol = this.titleRow.children[i]
            const show = titleCol.getAttribute("show")
            if (show === "xs" || !show) {
                xsCols.push(i)
            } else if (show === "sm") {
                smCols.push(i)
            } else if (show === "md") {
                mdCols.push(i)
            } else if (show === "lg") {
                lgCols.push(i)
            } else if (show === "xl") {
                xlCols.push(i)
            }
        }

        // TODO: grid-template-columns should be part of media query

        // next, generate the media styles
        // @param mediaWidth max media size for rule
        // @param templateCols is the column nrs whose width to include in the grid-template-columns
        const makeMediaStyle = (mediaWidth, templateCols) => {
            let templateWidths = ""
            if (templateCols.length === 0) {
                return ""
            }
            // 1. define the grid-template-columns widths to include in this media query
            for (let i = 0; i < this.titleRow.children.length; i++) {
                // If this column is visible then include its width
                if (templateCols.indexOf(i) !== -1) {
                    const titleCol = this.titleRow.children[i]
                    let width = titleCol.getAttribute("width")
                    if (!width) {
                        width = "auto"
                    }
                    templateWidths += " " + width
                }
            }

            // 2. create the media rule and add the grid-template-columns
            mediaStyle = "@media screen and (width < " + mediaWidth + ") {\n"
            mediaStyle += `  #${gridId} { grid-template-columns: ${templateWidths}; }\n`

            // 3. set all columns not included as display:none
            for (let i = 0; i < this.titleRow.children.length; i++) {
                // If this column is not visible then set display to none
                if (templateCols.indexOf(i) === -1) {
                    const colId = i + 1
                    mediaStyle += `\n  #${gridId} > li > *:nth-child(${colId}) {
                 display: none; 
              }`
                }
            }
            mediaStyle += "\n}\n"
            return mediaStyle
        }

        let gridTemplateCols = xsCols
        let mediaStyleRule = makeMediaStyle("576px", gridTemplateCols)
        if (mediaStyleRule) {
            styleSheet.insertRule(mediaStyleRule, 0);
        }
        gridTemplateCols = gridTemplateCols.concat(smCols)
        mediaStyleRule = makeMediaStyle("768px", gridTemplateCols)
        if (mediaStyleRule) {
            styleSheet.insertRule(mediaStyleRule, 0);
        }
        gridTemplateCols = gridTemplateCols.concat(mdCols)
        mediaStyleRule = makeMediaStyle("1024px", gridTemplateCols)
        if (mediaStyleRule) {
            styleSheet.insertRule(mediaStyleRule, 0);
        }
        gridTemplateCols = gridTemplateCols.concat(lgCols)
        mediaStyleRule = makeMediaStyle("1480px", gridTemplateCols)
        if (mediaStyleRule) {
            styleSheet.insertRule(mediaStyleRule, 0);
        }
        gridTemplateCols = gridTemplateCols.concat(xlCols)
        mediaStyleRule = makeMediaStyle("1920px", gridTemplateCols)
        if (mediaStyleRule) {
            styleSheet.insertRule(mediaStyleRule, 0);
        }
        mediaStyleRule = "#" + gridId + " { grid-template-columns:" + allColumnSizes + "}"
        styleSheet.insertRule(mediaStyleRule, 0);

        console.log("h-grid-table: adding rules for", gridId)
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
                const slotChildren = this.querySelectorAll(`[slot = '${slot.name}']`);
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
        this.titleRow = this.gridContainer.children[0]
        // Must have an id
        if (!this.id) {
            console.log("h-grid-table needs an id")
            this.id = "missing-id"
        }
        if (!this.gridContainer.id) {
            this.gridContainer.id = this.id + "-container"
        }

            this.createStyleSheet();

        const gridClassAttr = this.getAttribute(GRID_CLASS_ATTR)
        if (gridClassAttr) {
            let tokens = gridClassAttr.split(" ")
            this.gridContainer.classList.add(...tokens)
        }

        // only draggable items can be dragged (eg not the title row)
        this.gridContainer.addEventListener('dragstart', (ev) => {
            console.log("dragstart")
            // a cell in this row starts dragging
            if (ev.target.hasAttribute("draggable")) {
                this.draggedEl = this.getRow(this.gridContainer, ev.target)
                ev.dataTransfer.dropEffect = "move"
                ev.dataTransfer.effectAllowed = "move"
                ev.dataTransfer.setData("text/html", this.draggedEl.id)
            } else {
                // this is not a drag start point
                ev.stopPropagation()
            }
        })

        this.gridContainer.addEventListener('dragover', (ev) => {
            // console.log("dragover")
            // ??? why?
            ev.preventDefault();
            ev.stopPropagation()
        })
        this.gridContainer.addEventListener('dragend', (ev) => {
            console.log("dragend")
            if (this.draggedEl) {
                this.draggedEl.classList.remove('dragging');
            }
            // TODO: is this neccesary?
            for (let r of this.gridContainer.children) {
                r.classList.remove("hover")
            }
        })
        this.gridContainer.addEventListener('drop', (ev) => {
            console.log("drop")
            ev.stopPropagation()
            // move the dragged element after this element
            let targetRow = this.getRow(this.gridContainer, ev.target)
            if (this.draggedEl !== targetRow) {
                this.draggedEl.remove()
                targetRow.after(this.draggedEl)
            }
        })
        this.gridContainer.addEventListener('dragenter', (ev) => {
            // if this is a different target, clear the current target
            let currentTarget = this.getRow(this.gridContainer, ev.target)
            if (this.enterTarget && this.enterTarget !== currentTarget) {
                this.dragEnterCount = 0
                this.enterTarget.classList.remove("hover")
            }
            if (currentTarget) {
                console.log("dragenter; target", currentTarget.id)
                this.enterTarget = currentTarget
                this.dragEnterCounter++
                ev.stopPropagation()
                currentTarget.classList.add("hover")
            }
        })
        this.gridContainer.addEventListener('dragleave', (ev) => {
            this.dragEnterCounter--
            if (this.dragEnterCounter === 0) {
                let currentTarget = this.getRow(this.gridContainer, ev.target)
                if (currentTarget) {
                    console.log("dragleave; target", currentTarget.id)
                    ev.stopPropagation()
                    currentTarget.classList.remove("hover")
                }
                this.enterTarget = undefined
            }
        })
    }

// Get the container child row that has the given element
// This returns the row element that is a direct child of a container
    getRow(container, child) {
        let row = child
        while (row) {
            if (row.parentElement === container) {
                return row
            }
            row = row.parentElement
        }
        console.log("element is not a container child. id=", child.id)
        return undefined
    }


}

customElements.define('h-grid-table', HGridTable);
